package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"

	roundChan "github.com/0xl3mon/mksub-ng/round"
)

const (
	bufferSizeMB      = 10
	maxWorkingThreads = 100000
	numberOfFiles     = 1
	wordBatchSize     = 10000
	outputBufferSize  = 1000
)

var (
	inputDomains     []string
	wordSet          map[string]bool
	words            []string
	multipleWordSets []map[string]bool
	multipleWords    [][]string
	streamMode       bool
)

var (
	domain     string
	domainFile string
	wordlist   string
	regex      string
	level      int
	workers    int
	outputFile string
	silent     bool
	multiple   bool
	wordlists  []string
	prefix     string

	workerThreadMax = make(chan struct{}, maxWorkingThreads)
	done            = make(chan struct{})
	wg              sync.WaitGroup
	wgWrite         sync.WaitGroup
	robin           roundChan.RoundRobin
)

func readDomainFile() {
	inputFile, err := os.Open(domainFile)
	if err != nil {
		panic("Could not open file to read domains!")
	}
	defer inputFile.Close()

	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		inputDomains = append(inputDomains, strings.TrimSpace(scanner.Text()))
	}
}

func prepareDomains() {
	if domain == "" && domainFile == "" {
		fmt.Println("No domain input provided")
		os.Exit(1)
	}

	inputDomains = make([]string, 0)
	if domain != "" {
		// If prefix is provided and domain doesn't already contain FUZZ, apply it
		if prefix != "" && !strings.Contains(domain, "FUZZ") {
			inputDomains = append(inputDomains, prefix+"."+domain)
		} else {
			inputDomains = append(inputDomains, domain)
		}
	} else {
		if domainFile != "" {
			readDomainFile()
			// If prefix is provided, apply it only to domains that don't already contain FUZZ
			if prefix != "" {
				for i, d := range inputDomains {
					if !strings.Contains(d, "FUZZ") {
						inputDomains[i] = prefix + "." + d
					}
				}
			}
		}
	}
}

func readWordlistFile() {
	var reg *regexp.Regexp
	var err error
	if regex != "" {
		reg, err = regexp.Compile(regex)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	wordlistFile, err := os.Open(wordlist)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer wordlistFile.Close()

	// Check file size to determine if we should use streaming mode
	fileInfo, _ := wordlistFile.Stat()
	fileSize := fileInfo.Size()

	// Use streaming mode for files larger than 50MB
	if fileSize > 50*1024*1024 {
		streamMode = true
		return
	}

	wordSet = make(map[string]bool)
	scanner := bufio.NewScanner(wordlistFile)
	for scanner.Scan() {
		word := strings.ToLower(scanner.Text())
		word = strings.Trim(word, ".")
		if reg != nil {
			if !reg.Match([]byte(word)) {
				continue
			}
		}

		if word != "" {
			wordSet[word] = true
		}
	}

	for w := range wordSet {
		words = append(words, w)
	}
}

func readMultipleWordlistFiles() {
	var reg *regexp.Regexp
	var err error
	if regex != "" {
		reg, err = regexp.Compile(regex)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// Check if any wordlist is large enough to require streaming
	for _, wordlistPath := range wordlists {
		file, err := os.Open(wordlistPath)
		if err != nil {
			continue
		}
		fileInfo, _ := file.Stat()
		file.Close()
		if fileInfo.Size() > 50*1024*1024 {
			streamMode = true
			return
		}
	}

	multipleWordSets = make([]map[string]bool, len(wordlists))
	multipleWords = make([][]string, len(wordlists))

	for i, wordlistFile := range wordlists {
		file, err := os.Open(wordlistFile)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		defer file.Close()

		multipleWordSets[i] = make(map[string]bool)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			word := strings.ToLower(scanner.Text())
			word = strings.Trim(word, ".")
			if reg != nil {
				if !reg.Match([]byte(word)) {
					continue
				}
			}

			if word != "" {
				multipleWordSets[i][word] = true
			}
		}

		for w := range multipleWordSets[i] {
			multipleWords[i] = append(multipleWords[i], w)
		}
	}
}

func closeWriters(number int) {
	for i := 0; i < number; i++ {
		done <- struct{}{}
	}
}

func spawnWriters(number int) {
	for i := 0; i < number; i++ {
		var bf bytes.Buffer
		ch := make(chan string, 100000)

		var file *os.File
		var err error

		// Only create file if outputFile is specified
		if outputFile != "" {
			fileName := outputFile
			fileSplit := strings.Split(fileName, ".")
			if len(fileSplit) == 1 {
				fileName += ".txt"
			}
			if number > 1 {
				fileSplit = strings.Split(fileName, ".")
				extension := "." + fileSplit[len(fileSplit)-1]
				fileName = strings.TrimSuffix(fileName, extension) + "-" + strconv.Itoa(i) + extension
			}
			file, err = os.Create(fileName)
			if err != nil {
				fmt.Println(err)
				fmt.Println("Couldn't open file to write output!")
			}
		}
		// If no outputFile specified, file remains nil (stdout mode)

		wgWrite.Add(1)
		go write(file, &bf, &ch)

		if robin == nil {
			robin = roundChan.New(&ch)
			continue
		}
		robin.Add(&ch)
	}
}

func write(file *os.File, buffer *bytes.Buffer, ch *chan string) {
mainLoop:
	for {
		select {
		case <-done:
			for {
				if !writeOut(file, buffer, ch) {
					break
				}
			}
			if buffer.Len() > 0 {
				if file != nil {
					_, _ = file.WriteString(buffer.String())
					buffer.Reset()
				} else {
					// Output remaining buffer to stdout
					fmt.Print(buffer.String())
					buffer.Reset()
				}
			}
			break mainLoop
		default:
			writeOut(file, buffer, ch)
		}
	}
	wgWrite.Done()
}
func writeOut(file *os.File, buffer *bytes.Buffer, outputChannel *chan string) bool {
	select {
	case s := <-*outputChannel:
		buffer.WriteString(s)
		// Smaller buffer for better streaming performance
		if buffer.Len() >= bufferSizeMB*1024*1024 {
			if file != nil {
				_, _ = file.WriteString(buffer.String())
				buffer.Reset()
			} else {
				// Output to stdout when no file specified
				fmt.Print(buffer.String())
				buffer.Reset()
			}
		}
		return true
	default:
		return false
	}
}

func combo(_comb string, level int, wg *sync.WaitGroup, wt *chan struct{}) {
	defer wg.Done()
	workerThreadMax <- struct{}{}

	if strings.Count(_comb, ".") > 1 {
		if !silent {
			fmt.Print(_comb + "\n")
		}
		*robin.Next() <- _comb + "\n"
	}

	var nextLevelWaitGroup sync.WaitGroup
	if level > 1 {
		nextLevelWt := make(chan struct{}, workers)
		for _, c := range words {
			nextLevelWaitGroup.Add(1)
			nextLevelWt <- struct{}{}
			go combo(c+"."+_comb, level-1, &nextLevelWaitGroup, &nextLevelWt)
		}
	} else {
		for _, c := range words {
			if !silent {
				fmt.Print(c + "." + _comb + "\n")
			}
			*robin.Next() <- c + "." + _comb + "\n"
		}
	}

	nextLevelWaitGroup.Wait()
	<-workerThreadMax
	<-*wt
}

func fuzzCombo(domain string) {
	if streamMode {
		if multiple {
			generateFuzzCombinationsMultipleStream(domain)
		} else {
			generateFuzzCombinationsStream(domain)
		}
	} else {
		if multiple {
			generateFuzzCombinationsMultiple(domain)
		} else {
			generateFuzzCombinations(domain)
		}
	}
}

func generateFuzzCombinations(domain string) {
	if strings.Contains(domain, "FUZZ") {
		fuzzCount := strings.Count(domain, "FUZZ")
		if fuzzCount == 1 {
			for _, word := range words {
				result := strings.Replace(domain, "FUZZ", word, -1)
				if !silent {
					fmt.Print(result + "\n")
				}
				*robin.Next() <- result + "\n"
			}
		} else {
			generateMultipleFuzzCombinations(domain, words)
		}
	}
}

func generateMultipleFuzzCombinations(domain string, wordList []string) {
	for _, word1 := range wordList {
		for _, word2 := range wordList {
			result := domain
			result = strings.Replace(result, "FUZZ", word1, 1)
			for strings.Contains(result, "FUZZ") {
				result = strings.Replace(result, "FUZZ", word2, 1)
			}
			if !silent {
				fmt.Print(result + "\n")
			}
			*robin.Next() <- result + "\n"
		}
	}
}

func generateFuzzCombinationsMultiple(domain string) {
	placeholders := extractNumberedPlaceholders(domain)
	if len(placeholders) == 0 {
		return
	}

	generateNumberedFuzzCombinations(domain, placeholders, 0, make(map[string]string))
}

func extractNumberedPlaceholders(domain string) []string {
	re := regexp.MustCompile(`FUZZ(\d+)`)
	matches := re.FindAllStringSubmatch(domain, -1)
	placeholderSet := make(map[string]bool)
	var placeholders []string

	for _, match := range matches {
		placeholder := "FUZZ" + match[1]
		if !placeholderSet[placeholder] {
			placeholderSet[placeholder] = true
			placeholders = append(placeholders, placeholder)
		}
	}
	return placeholders
}

func generateNumberedFuzzCombinations(domain string, placeholders []string, index int, replacements map[string]string) {
	if index >= len(placeholders) {
		result := domain
		for placeholder, word := range replacements {
			result = strings.ReplaceAll(result, placeholder, word)
		}
		if !silent {
			fmt.Print(result + "\n")
		}
		*robin.Next() <- result + "\n"
		return
	}

	placeholder := placeholders[index]
	placeholderNum, _ := strconv.Atoi(strings.TrimPrefix(placeholder, "FUZZ"))
	wordlistIndex := placeholderNum - 1

	if wordlistIndex >= 0 && wordlistIndex < len(multipleWords) {
		for _, word := range multipleWords[wordlistIndex] {
			replacements[placeholder] = word
			generateNumberedFuzzCombinations(domain, placeholders, index+1, replacements)
		}
	}
}

// Streaming functions for large wordlists
func generateFuzzCombinationsStream(domain string) {
	var reg *regexp.Regexp
	var err error
	if regex != "" {
		reg, err = regexp.Compile(regex)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	wordlistFile, err := os.Open(wordlist)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer wordlistFile.Close()

	scanner := bufio.NewScanner(wordlistFile)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	seenWords := make(map[string]bool)
	outputBuffer := make([]string, 0, outputBufferSize)

	for scanner.Scan() {
		word := strings.ToLower(scanner.Text())
		word = strings.Trim(word, ".")

		if word == "" || seenWords[word] {
			continue
		}

		if reg != nil && !reg.Match([]byte(word)) {
			continue
		}

		seenWords[word] = true

		if strings.Contains(domain, "FUZZ") {
			fuzzCount := strings.Count(domain, "FUZZ")
			if fuzzCount == 1 {
				result := strings.Replace(domain, "FUZZ", word, -1)
				outputBuffer = append(outputBuffer, result+"\n")
			} else {
				// For multiple FUZZ, we need to generate combinations
				generateMultipleFuzzForWord(domain, word, &outputBuffer)
			}

			if len(outputBuffer) >= outputBufferSize {
				flushOutputBuffer(&outputBuffer)
			}
		}
	}

	if len(outputBuffer) > 0 {
		flushOutputBuffer(&outputBuffer)
	}
}

func generateMultipleFuzzForWord(domain, word string, outputBuffer *[]string) {
	// For streaming mode with multiple FUZZ, we generate combinations with the current word
	result := domain
	result = strings.Replace(result, "FUZZ", word, 1)
	for strings.Contains(result, "FUZZ") {
		result = strings.Replace(result, "FUZZ", word, 1)
	}
	*outputBuffer = append(*outputBuffer, result+"\n")
}

func generateFuzzCombinationsMultipleStream(domain string) {
	placeholders := extractNumberedPlaceholders(domain)
	if len(placeholders) == 0 {
		return
	}

	// For streaming mode with multiple wordlists, we need a different approach
	// We'll process the first wordlist and for each word, stream through others
	generateNumberedFuzzCombinationsStream(domain, placeholders)
}

func generateNumberedFuzzCombinationsStream(domain string, placeholders []string) {
	var reg *regexp.Regexp
	var err error
	if regex != "" {
		reg, err = regexp.Compile(regex)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// For simplicity in streaming mode, we'll load smaller wordlists into memory
	// and stream only the largest one
	largestIndex := 0
	largestSize := int64(0)

	for i, wordlistPath := range wordlists {
		file, err := os.Open(wordlistPath)
		if err != nil {
			continue
		}
		fileInfo, _ := file.Stat()
		file.Close()
		if fileInfo.Size() > largestSize {
			largestSize = fileInfo.Size()
			largestIndex = i
		}
	}

	// Load smaller wordlists into memory
	smallWordlists := make([][]string, len(wordlists))
	for i, wordlistPath := range wordlists {
		if i == largestIndex {
			continue
		}

		file, err := os.Open(wordlistPath)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		wordSet := make(map[string]bool)
		for scanner.Scan() {
			word := strings.ToLower(scanner.Text())
			word = strings.Trim(word, ".")
			if word != "" && (reg == nil || reg.Match([]byte(word))) {
				wordSet[word] = true
			}
		}

		for word := range wordSet {
			smallWordlists[i] = append(smallWordlists[i], word)
		}
	}

	// Stream through the largest wordlist
	largestFile, err := os.Open(wordlists[largestIndex])
	if err != nil {
		return
	}
	defer largestFile.Close()

	scanner := bufio.NewScanner(largestFile)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	seenWords := make(map[string]bool)
	outputBuffer := make([]string, 0, outputBufferSize)

	for scanner.Scan() {
		word := strings.ToLower(scanner.Text())
		word = strings.Trim(word, ".")

		if word == "" || seenWords[word] {
			continue
		}

		if reg != nil && !reg.Match([]byte(word)) {
			continue
		}

		seenWords[word] = true

		// Generate combinations with this word from the largest wordlist
		generateStreamCombinations(domain, placeholders, largestIndex, word, smallWordlists, &outputBuffer)

		if len(outputBuffer) >= outputBufferSize {
			flushOutputBuffer(&outputBuffer)
		}
	}

	if len(outputBuffer) > 0 {
		flushOutputBuffer(&outputBuffer)
	}
}

func generateStreamCombinations(domain string, placeholders []string, largestIndex int, largestWord string, smallWordlists [][]string, outputBuffer *[]string) {
	replacements := make(map[string]string)

	// Set the word from the largest wordlist
	largestPlaceholder := fmt.Sprintf("FUZZ%d", largestIndex+1)
	replacements[largestPlaceholder] = largestWord

	// Generate combinations with other wordlists
	generateStreamCombinationsRecursive(domain, placeholders, 0, replacements, smallWordlists, outputBuffer)
}

func generateStreamCombinationsRecursive(domain string, placeholders []string, index int, replacements map[string]string, smallWordlists [][]string, outputBuffer *[]string) {
	if index >= len(placeholders) {
		result := domain
		for placeholder, word := range replacements {
			result = strings.ReplaceAll(result, placeholder, word)
		}
		*outputBuffer = append(*outputBuffer, result+"\n")
		return
	}

	placeholder := placeholders[index]
	if _, exists := replacements[placeholder]; exists {
		// Already set, move to next
		generateStreamCombinationsRecursive(domain, placeholders, index+1, replacements, smallWordlists, outputBuffer)
		return
	}

	placeholderNum, _ := strconv.Atoi(strings.TrimPrefix(placeholder, "FUZZ"))
	wordlistIndex := placeholderNum - 1

	if wordlistIndex >= 0 && wordlistIndex < len(smallWordlists) && len(smallWordlists[wordlistIndex]) > 0 {
		for _, word := range smallWordlists[wordlistIndex] {
			replacements[placeholder] = word
			generateStreamCombinationsRecursive(domain, placeholders, index+1, replacements, smallWordlists, outputBuffer)
			delete(replacements, placeholder)
		}
	}
}

func flushOutputBuffer(outputBuffer *[]string) {
	for _, result := range *outputBuffer {
		if !silent {
			fmt.Print(result)
		}
		*robin.Next() <- result
	}
	*outputBuffer = (*outputBuffer)[:0]
}

func main() {
	flag.StringVar(&domain, "d", "", "Input domain")
	flag.StringVar(&domainFile, "df", "", "Input domain file, one domain per line")
	flag.StringVar(&wordlist, "w", "", "Wordlist file")
	flag.StringVar(&regex, "r", "", "Regex to filter words from wordlist file")
	flag.IntVar(&level, "l", 1, "Subdomain level to generate")
	flag.StringVar(&outputFile, "o", "", "Output file (stdout will be used when omitted)")
	flag.IntVar(&workers, "t", 100, "Number of threads for every subdomain level")
	flag.BoolVar(&silent, "silent", true, "Skip writing generated subdomains to stdout (faster)")
	flag.BoolVar(&multiple, "multiple", false, "Enable multiple wordlist mode with numbered FUZZ placeholders")
	flag.StringVar(&prefix, "prefix", "", "Prefix pattern to apply to domains (e.g., 'FUZZ', 'FUZZ2-FUZZ1')")

	// Custom flag parsing for multiple -w flags
	flag.Parse()

	// Handle help flag
	for _, arg := range os.Args {
		if arg == "-?" || arg == "-h" || arg == "--help" {
			flag.Usage()
			os.Exit(0)
		}
	}

	// Parse multiple -w flags
	for i, arg := range os.Args {
		if arg == "-w" && i+1 < len(os.Args) {
			wordlists = append(wordlists, os.Args[i+1])
		}
	}

	go func() {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
		<-signalChannel

		fmt.Println("Program interrupted, exiting...")
		os.Exit(0)
	}()

	if level <= 0 || workers <= 0 {
		fmt.Println("Subdomain level and number of threads must be positive integers!")
		os.Exit(0)
	}

	if outputFile == "" {
		silent = false
	}

	prepareDomains()

	// Check if any domain contains FUZZ or if prefix mode is enabled
	hasFuzz := false
	for _, d := range inputDomains {
		if strings.Contains(d, "FUZZ") {
			hasFuzz = true
			break
		}
	}

	// Enable fuzz mode if prefix is provided
	if prefix != "" {
		hasFuzz = true

		// Auto-detect if multiple mode is needed based on prefix pattern
		if strings.Contains(prefix, "FUZZ1") || strings.Contains(prefix, "FUZZ2") || strings.Contains(prefix, "FUZZ3") || strings.Contains(prefix, "FUZZ4") {
			if !multiple {
				multiple = true // Auto-enable multiple mode
			}
		}
	}

	// Validate prefix usage
	if prefix != "" {
		if multiple && len(wordlists) == 0 {
			fmt.Println("--prefix mode with multiple FUZZ placeholders requires wordlist files (-w)")
			os.Exit(1)
		}
		if !multiple && wordlist == "" && len(wordlists) == 0 {
			fmt.Println("--prefix mode requires at least one wordlist file (-w)")
			os.Exit(1)
		}
		// If using single wordlist mode with prefix, use the first wordlist as the main wordlist
		if !multiple && wordlist == "" && len(wordlists) > 0 {
			wordlist = wordlists[0]
		}
	}

	// Validate FUZZ mode requirements (when domains contain FUZZ but no prefix is used)
	if hasFuzz && prefix == "" {
		if multiple && len(wordlists) == 0 {
			fmt.Println("FUZZ mode with multiple placeholders requires wordlist files (-w)")
			os.Exit(1)
		}
		if !multiple && wordlist == "" && len(wordlists) == 0 {
			fmt.Println("FUZZ mode requires at least one wordlist file (-w)")
			os.Exit(1)
		}
		// If using single wordlist mode with FUZZ patterns, use the first wordlist as the main wordlist
		if !multiple && wordlist == "" && len(wordlists) > 0 {
			wordlist = wordlists[0]
		}
	}

	if hasFuzz {
		if multiple {
			if len(wordlists) == 0 {
				fmt.Println("Multiple mode requires at least one wordlist file")
				os.Exit(1)
			}
			readMultipleWordlistFiles()
		} else {
			if wordlist == "" {
				fmt.Println("FUZZ mode requires a wordlist file")
				os.Exit(1)
			}
			readWordlistFile()
		}
		spawnWriters(numberOfFiles)

		for _, d := range inputDomains {
			fuzzCombo(d)
		}
	} else {
		readWordlistFile()
		spawnWriters(numberOfFiles)

		for _, d := range inputDomains {
			wg.Add(1)
			wt := make(chan struct{}, 1)
			wt <- struct{}{}
			go combo(d, level, &wg, &wt)
		}

		wg.Wait()
	}

	closeWriters(numberOfFiles)
	wgWrite.Wait()
}
