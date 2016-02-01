package main

import (
    "bufio"
    "os"
    "strings"
    "fmt"
    "regexp"
)

const (
	SyllableFilename = "./cmudict-0.7b"
)

type Word struct {
    Name           string
    FragmentString string
    Fragments      []string
    NumSyllables   int
    FinalSyllable  string
}

func readSyllables(filename string) (*map[string]Word) {

	words := map[string]Word{}
	countFragments      := 0
	countSyllables      := 0
    syllableRegexp      := regexp.MustCompile(`^[A-Z]+\d+$`)
    finalSyllableRegexp := regexp.MustCompile(`([A-Z]+\d+(?:[^\d]*))$`)

    // Open the file.
    f, _ := os.Open(filename)
    // Create a new Scanner for the file.
    scanner := bufio.NewScanner(f)
    // Loop over all lines in the file
    for scanner.Scan() {
		line := scanner.Text()
		if ! strings.HasPrefix(line, ";;;") {
			nameAndRemainder := strings.Split(line, "  ")
			name             := nameAndRemainder[0]
			remainder        := nameAndRemainder[1]
			fragments        := strings.Split(remainder, " ")

			numSyllables := 0
			for _,f := range fragments {
				if syllableRegexp.FindStringSubmatch(f) != nil {
					numSyllables = numSyllables + 1
	    		}
	    	}

	    	if numSyllables == 0 {
	    		fmt.Println("WARNING: no syllables found for name=", name) 
	    	}

	    	matches := finalSyllableRegexp.FindStringSubmatch(remainder)
	    	finalSyllable := ""
	    	if matches != nil {
	    		finalSyllable = matches[1]
	    	} else {
	    		fmt.Println("WARNING: no final syllable found for name=", name) 
	    	}

	    	countSyllables = countSyllables + numSyllables
			countFragments = countFragments + len(fragments)
			words[name] = Word{
				Name:           name,
				FragmentString: remainder,
				Fragments:      fragments,
				NumSyllables:   numSyllables,
				FinalSyllable:  finalSyllable,
			}
		}
    }

    fmt.Println("num fragments = ", countFragments, ", num syllables = ", countSyllables) 

    return &words
}

func processFinalSyllables(words *map[string]Word) (*map[string][]*Word) {
	finalSyllables := map[string][]*Word{}

	for _,word := range *words {
		fs := word.FinalSyllable
		var rhymingWords []*Word

		rhymingWords, ok := finalSyllables[fs]
		if ! ok {
			rhymingWords = []*Word{}
		}

		finalSyllables[fs] = append( rhymingWords, &word )
	}

	// for fs,rw := range finalSyllables {
	// 	fmt.Println("fs: ", fs, ", num = ", len(rw))
	// }

	return &finalSyllables
}

type Syllabi struct {
    Words          *map[string]Word
    FinalOnes      *map[string][]*Word
    SourceFilename string
}

func ConstructSyllabi(sourceFilename string) (*Syllabi){
	words          := readSyllables(SyllableFilename)
	finalSyllables := processFinalSyllables(words)

	syllabi := Syllabi{
		Words:          words,
		FinalOnes:      finalSyllables,
		SourceFilename: SyllableFilename,
	}

	return &syllabi
}

func main() {
	syllabi := ConstructSyllabi(SyllableFilename)

    fmt.Println("num words = ", len(*syllabi.Words) ) 
	fmt.Println("num unique final syllables = ", len(*syllabi.FinalOnes))
}
