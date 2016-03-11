package main

import (
	"github.com/railsagainstignorance/alignment/Godeps/_workspace/src/github.com/joho/godotenv"
	"html/template"
	"net/http"
	"os"
    "sort"
    "regexp"
    "fmt"
    // "strings"
    "strconv"
    "github.com/railsagainstignorance/alignment/align"
    // "github.com/railsagainstignorance/alignment/sapi"
    "github.com/railsagainstignorance/alignment/rhyme"
    "github.com/railsagainstignorance/alignment/article"
    "github.com/railsagainstignorance/alignment/content"
    "github.com/railsagainstignorance/alignment/ontology"
)

// compile all templates and cache them
var templates = template.Must(template.ParseGlob("templates/*"))

func templateExecuter( w http.ResponseWriter, pageName string, data interface{} ){
    err := templates.ExecuteTemplate(w, pageName, data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }    
}

func alignFormHandler(w http.ResponseWriter, r *http.Request) {
    templateExecuter( w, "alignPage", nil )
}

func alignHandler(w http.ResponseWriter, r *http.Request) {
    p := align.Search( r.FormValue("text"), r.FormValue("source") )
    templateExecuter( w, "alignedPage", p )
}

type ResultItemWithRhymeAndMeter struct {
    ResultItem    *(content.Article)
    RhymeAndMeter *(rhyme.RhymeAndMeter)
}

type SearchResultWithRhymeAndMeterList struct {
    SearchResult *(content.SearchResponse)
    ResultItemsWithRhymeAndMeterList []*ResultItemWithRhymeAndMeter
    MatchMeter string
    EmphasisRegexp *(regexp.Regexp)
    EmphasisRegexpString string
    KnownUnknowns *[]string
    PhraseWordsRegexpString string
    Text string
    Source string
    TitleOnlyChecked string
    AnyChecked       string    
}

type RhymedResultItems []*ResultItemWithRhymeAndMeter

func (rri RhymedResultItems) Len()          int  { return len(rri) }
func (rri RhymedResultItems) Swap(i, j int)      { rri[i], rri[j] = rri[j], rri[i] }
func (rri RhymedResultItems) Less(i, j int) bool { return rri[i].RhymeAndMeter.FinalSyllable > rri[j].RhymeAndMeter.FinalSyllable }

var syllabi = rhyme.ConstructSyllabi(&[]string{"rhyme/cmudict-0.7b", "rhyme/cmudict-0.7b_my_additions"})

func meterHandler(w http.ResponseWriter, r *http.Request) {
    // searchParams := sapi.SearchParams{
    //     Text:   r.FormValue("text"),
    //     Source: r.FormValue("source"),
    // }
    // sapiResult := sapi.Search( searchParams )

    text := r.FormValue("text")
    source := r.FormValue("source")

    var textForSearch string

    if source != "title-only" {
        source = "keyword"
        textForSearch = `\"` + text + `\"`
    } else {
        textForSearch = text
    }

    var (
        titleOnlyChecked string = ""
        anyChecked       string = ""
    )

    if source == "title-only" {
        titleOnlyChecked = "checked"
    } else {
        anyChecked       = "checked"
    }



    sRequest := &content.SearchRequest {
        QueryType: source,
        QueryText: textForSearch,
        MaxArticles: 100,
        MaxDurationMillis: 3000,
        SearchOnly: true, // i.e. don't bother looking up articles
    }

    sapiResult := content.Search( sRequest )

    matchMeter     := r.FormValue("meter")
    if matchMeter == "" {
        matchMeter = rhyme.DefaultMeter
    }

    emphasisRegexp, _ := rhyme.ConvertToEmphasisPointsStringRegexp(matchMeter)

    riwfsList := []*ResultItemWithRhymeAndMeter{}

    for _, item := range *(sapiResult.Articles) {
        var phrase string

        if source == "title-only" {
            phrase = item.Title
        } else {
            phrase = item.Excerpt
        }

        rams := syllabi.RhymeAndMetersOfPhrase(phrase, emphasisRegexp)

        if rams != nil {
            for _,ram := range *rams {
                if ram.EmphasisRegexpMatch2 != "" {
                    riwfs := ResultItemWithRhymeAndMeter{
                        ResultItem:    item,
                        RhymeAndMeter: ram,
                    }

                    riwfsList = append( riwfsList, &riwfs)            
                }
            }
        }
    }

    sort.Sort(RhymedResultItems(riwfsList))

    srwfs := SearchResultWithRhymeAndMeterList{
        SearchResult: sapiResult,
        ResultItemsWithRhymeAndMeterList:  riwfsList,
        MatchMeter:           matchMeter,
        EmphasisRegexp:       emphasisRegexp,
        EmphasisRegexpString: emphasisRegexp.String(),
        KnownUnknowns:        syllabi.KnownUnknowns(),
        PhraseWordsRegexpString: syllabi.PhraseWordsRegexpString,
        Text: text,
        Source: source,
        TitleOnlyChecked: titleOnlyChecked,
        AnyChecked: anyChecked,
    }

    templateExecuter( w, "meteredPage", &srwfs )
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
    uuid  := r.FormValue("uuid")
    meter := r.FormValue("meter")

    p := article.GetArticleWithSentencesAndMeter(uuid, meter, syllabi )
    templateExecuter( w, "articlePage", p )
}

func detailHandler(w http.ResponseWriter, r *http.Request) {
    phrase         := r.FormValue("phrase")
    sentences      := []string{ phrase }
    meter          := r.FormValue("meter")
    rams           := article.FindRhymeAndMetersInSentences( &sentences, meter, syllabi )
    meterRegexp,_  := rhyme.ConvertToEmphasisPointsStringRegexp(meter)

    type PhraseDetails struct {
        Phrase string
        Sentences *[]string
        Meter string
        MeterRegexp *regexp.Regexp
        RhymeAndMeters *[]*rhyme.RhymeAndMeter
        KnownUnknowns *[]string
        EmphasisPointsDetails *rhyme.EmphasisPointsDetails
    }

    pd := PhraseDetails{
        Phrase:         phrase,
        Sentences:      &sentences,
        Meter:          meter,
        MeterRegexp:    meterRegexp,
        RhymeAndMeters: rams,
        KnownUnknowns:  syllabi.KnownUnknowns(),
        EmphasisPointsDetails: syllabi.FindAllEmphasisPointsDetails(phrase),
    }

    templateExecuter( w, "detailPage", pd )
}

func ontologyHandler(w http.ResponseWriter, r *http.Request) {
    ontologyName  := r.FormValue("ontology")
    ontologyvalue := r.FormValue("value")
    meter  := r.FormValue("meter")

    maxArticles := 10
    if r.FormValue("max") != "" {
        i, err := strconv.Atoi(r.FormValue("max"))
        if err == nil {
            maxArticles = i
        }
    }
    
    maxMillis   := 3000

    details, containsHaikus := ontology.GetDetails(syllabi, ontologyName, ontologyvalue, meter, maxArticles, maxMillis)

    if containsHaikus {
        templateExecuter( w, "ontologyHaikuPage", details )
    } else {
        templateExecuter( w, "ontologyPage", details )
    }
}

type ResultWithFinalSyllable struct {
    *content.Article
    FinalSyllableAZ string
    FirstOfNewRhyme bool
}

func (f *ResultWithFinalSyllable) SetFirstOfNewRhyme(val bool) {
    f.FirstOfNewRhyme = val
}

type ResultsWithFinalSyllable []*ResultWithFinalSyllable

func (rwfs ResultsWithFinalSyllable) Len()          int  { return len(rwfs) }
func (rwfs ResultsWithFinalSyllable) Swap(i, j int)      { rwfs[i], rwfs[j] = rwfs[j], rwfs[i] }
func (rwfs ResultsWithFinalSyllable) Less(i, j int) bool { return rwfs[i].FinalSyllableAZ > rwfs[j].FinalSyllableAZ }

type FSandCount struct {
    FinalSyllable string
    Count int
}
type FSandCounts []*FSandCount

func (fsc FSandCounts) Len()          int  { return len(fsc) }
func (fsc FSandCounts) Swap(i, j int)      { fsc[i], fsc[j] = fsc[j], fsc[i] }
func (fsc FSandCounts) Less(i, j int) bool { return (fsc[j].FinalSyllable == "") || ((fsc[i].FinalSyllable != "") && (fsc[i].Count > fsc[j].Count)) }

func rhymeHandler(w http.ResponseWriter, r *http.Request) {
    text := r.FormValue("text")
    // searchParams := sapi.SearchParams{
    //     Text:   text,
    //     Source: "any",
    // }

    // sapiResult := sapi.Search( searchParams )

    sRequest := &content.SearchRequest {
        QueryType: "title",
        QueryText: text,
        MaxArticles: 100,
        MaxDurationMillis: 3000,
        SearchOnly: true, // i.e. don't bother looking up articles
    }

    sapiResult := content.Search( sRequest )

    finalSyllablesMap := map[string][]*ResultWithFinalSyllable{}

    for _, item := range *(sapiResult.Articles) {
        phrase := item.Title
        fs     := syllabi.FinalSyllableOfPhrase(phrase)
        fsAZ   := rhyme.KeepAZString( fs )
        rwfs   := &ResultWithFinalSyllable{
            item,
            fsAZ,
            false,
        }
        // rwfsList = append( rwfsList, rwfs )
        if _, ok := finalSyllablesMap[fsAZ]; !ok {
            finalSyllablesMap[fsAZ] = []*ResultWithFinalSyllable{}
        }

        finalSyllablesMap[fsAZ] = append(finalSyllablesMap[fsAZ], rwfs)
    }

    fsCounts := []*FSandCount{}

    for fs, list := range finalSyllablesMap {
        fsCounts = append(fsCounts, &FSandCount{fs, len(list)} )
    }

    sort.Sort(FSandCounts(fsCounts))
    rwfsList := []*ResultWithFinalSyllable{}

    for _, fsc := range fsCounts {
        fsList := finalSyllablesMap[fsc.FinalSyllable]
        for i,rwfs := range fsList {
            isFirst := (i == 0)
            rwfs.SetFirstOfNewRhyme( isFirst )
            rwfsList = append( rwfsList, rwfs)
        }
    }

    type Results struct {
        Text string
        ResultsWithFinalSyllable *[]*ResultWithFinalSyllable
    }

    p := Results{
        Text: text,
        ResultsWithFinalSyllable: &rwfsList,
    }

    templateExecuter( w, "rhymedPage", p )
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
    if port=="" {
        port = "8080"
    }

	http.HandleFunc("/",        log(alignFormHandler))
    http.HandleFunc("/align",   log(alignHandler))
    http.HandleFunc("/meter",   log(meterHandler))
    http.HandleFunc("/article", log(articleHandler))
    http.HandleFunc("/detail",  log(detailHandler))
    http.HandleFunc("/ontology", log(ontologyHandler))
    http.HandleFunc("/rhyme",   log(rhymeHandler))

	http.ListenAndServe(":"+string(port), nil)
}
