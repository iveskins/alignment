package main

import (
	"fmt"
	"github.com/railsagainstignorance/alignment/Godeps/_workspace/src/github.com/Financial-Times/ft-s3o-go/s3o"
	"github.com/railsagainstignorance/alignment/Godeps/_workspace/src/github.com/joho/godotenv"
	"github.com/railsagainstignorance/alignment/align"
	"github.com/railsagainstignorance/alignment/article"
	"github.com/railsagainstignorance/alignment/ontology"
	"github.com/railsagainstignorance/alignment/rhyme"
	"html/template"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

// compile all templates and cache them
var templates = template.Must(template.ParseGlob("templates/*"))

// construct the syllable monster
var syllabi = rhyme.ConstructSyllabi(&[]string{"rhyme/cmudict-0.7b", "rhyme/cmudict-0.7b_my_additions"})

func templateExecuter(w http.ResponseWriter, pageName string, data interface{}) {
	err := templates.ExecuteTemplate(w, pageName, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func alignFormHandler(w http.ResponseWriter, r *http.Request) {
	templateExecuter(w, "alignPage", nil)
}

func alignHandler(w http.ResponseWriter, r *http.Request) {
	p := align.Search(r.FormValue("text"), r.FormValue("source"))
	templateExecuter(w, "alignedPage", p)
}

func detailHandler(w http.ResponseWriter, r *http.Request) {
	phrase := r.FormValue("phrase")
	sentences := []string{phrase}
	meter := r.FormValue("meter")
	rams := article.FindRhymeAndMetersInSentences(&sentences, meter, syllabi)
	meterRegexp, _ := rhyme.ConvertToEmphasisPointsStringRegexp(meter)

	type PhraseDetails struct {
		Phrase                string
		Sentences             *[]string
		Meter                 string
		MeterRegexp           *regexp.Regexp
		RhymeAndMeters        *[]*rhyme.RhymeAndMeter
		KnownUnknowns         *[]string
		EmphasisPointsDetails *rhyme.EmphasisPointsDetails
	}

	pd := PhraseDetails{
		Phrase:                phrase,
		Sentences:             &sentences,
		Meter:                 meter,
		MeterRegexp:           meterRegexp,
		RhymeAndMeters:        rams,
		KnownUnknowns:         syllabi.KnownUnknowns(),
		EmphasisPointsDetails: syllabi.FindAllEmphasisPointsDetails(phrase),
	}

	templateExecuter(w, "detailPage", pd)
}

func ontologyHandler(w http.ResponseWriter, r *http.Request) {
	ontologyName := r.FormValue("ontology")
	ontologyValue := r.FormValue("value")
	meter := r.FormValue("meter")

	maxArticles := 10
	if r.FormValue("max") != "" {
		i, err := strconv.Atoi(r.FormValue("max"))
		if err == nil {
			maxArticles = i
		}
	}

	maxMillis := 3000

	details, containsHaikus := ontology.GetDetails(syllabi, ontologyName, ontologyValue, meter, maxArticles, maxMillis)

	if containsHaikus {
		templateExecuter(w, "ontologyHaikuPage", details)
	} else {
		templateExecuter(w, "ontologyPage", details)
	}
}

func log(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("REQUEST URL: ", r.URL)
		fn(w, r)
	}
}

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", log(alignFormHandler))
	http.HandleFunc("/align", log(alignHandler))
	// http.HandleFunc("/article", log(ontologyHandler))
	http.HandleFunc("/detail", log(detailHandler))
	http.Handle("/ontology", s3o.Handler(http.HandlerFunc(log(ontologyHandler))))

	http.ListenAndServe(":"+string(port), nil)
}
