package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

const dbFile string = "./realestatedb.json"

// Residence Details
type Residence struct {
	Name        string
	URL         string
	City        string
	ProjectSize string
	NumUnits    string
	PhoneNo     string
}

func itemExists(slice interface{}, item interface{}) bool {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		fmt.Println("Invalid data-type")
		return false
	}

	for i := 0; i < s.Len(); i++ {
		if s.Index(i).Interface() == item {
			return true
		}
	}

	return false
}

func inExcludeList(el []string, url string) bool {
	for _, elstr := range el {
		if strings.Index(url, elstr) != -1 {
			//fmt.Printf("Url : %s matches %s in exclude list\n", url, elstr)
			return true
		}
	}
	return false
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Collecting Prestige Residence Details
func prestigeResidenceDetails(ul []string, city string, prp *[]Residence) {
	var rName string
	var rProjSize string
	var rNumUnits string
	var rPhoneNo string
	var res Residence

	c := colly.NewCollector()

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*www.prestigeconstructions.com*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnHTML(`ul[class=list-two-col]`, func(e *colly.HTMLElement) {
		ckDevSize := strings.Contains(e.Text, "Development Size")
		ckNumUnit := strings.Contains(e.Text, "Number of Units")
		ckContact := strings.Contains(e.Text, "Contact No")
		if ckDevSize && ckNumUnit && ckContact {
			fmt.Println("Found Details")
			ch := e.DOM.Children()
			for i := range ch.Nodes {
				s := ch.Eq(i).Text()
				words := strings.FieldsFunc(s, func(r rune) bool {
					return strings.ContainsRune(":", r)
				})
				// fmt.Printf("%s : %s\n", words[0], words[1])
				if ok := strings.Contains(s, "Development Size"); ok {
					rProjSize = words[1]
				} else if ok := strings.Contains(s, "Number of Units"); ok {
					rNumUnits = words[1]
				} else if ok := strings.Contains(s, "Contact No"); ok {
					rPhoneNo = words[1]
				}
			}
		}
	})

	c.OnScraped(func(r *colly.Response) {
		if r.StatusCode != 200 {
			fmt.Println("OnScraped, StatusCode = ", r.StatusCode)
		}
		//fmt.Println("OnScraped, Body = ", string(r.Body))
	})

	for _, url := range ul {
		path := strings.FieldsFunc(url, func(r rune) bool {
			return strings.ContainsRune("/", r)
		})
		rName = path[len(path)-1]
		rProjSize = ""
		rNumUnits = ""
		rPhoneNo = ""
		//fmt.Printf("Details for %s\n", path[len(path)-1])
		c.Visit(url)
		res = Residence{
			Name:        rName,
			URL:         url,
			City:        city,
			ProjectSize: rProjSize,
			NumUnits:    rNumUnits,
			PhoneNo:     rPhoneNo,
		}
		*prp = append(*prp, res)
	}
}

// Collecting Prestige Residences
func prestigeResidenceCollector(url string, el []string) []string {
	var rp []string

	c := colly.NewCollector()

	// Limit the number of threads started by colly to two
	// when visiting links which domains' matches "*httpbin.*" glob
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*www.prestigeconstructions.com*",
		Parallelism: 1,
		Delay:       1 * time.Second,
	})

	/***
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	***/

	// check every href
	c.OnHTML(`a[href]`, func(e *colly.HTMLElement) {
		projURL := e.Request.AbsoluteURL(e.Attr("href"))
		// Check if URL is about a real estate project
		if strings.Index(projURL, "projects/") != -1 {

			// Check if it already added or in exclude list
			found := itemExists(rp, projURL)
			exclude := inExcludeList(el, projURL)
			if !found && !exclude {
				fmt.Println("Adding : ", projURL)
				rp = append(rp, projURL)
				e.Request.Visit(projURL)
			}
		}
	})

	// Search for multiple pages
	c.OnHTML(".pagination", func(e *colly.HTMLElement) {
		//fmt.Println("Hit onHTML .pagination")
		//fmt.Printf("e.Name: %s, e.Text: %s\n", e.Name, e.Text)
		//fmt.Printf("e.Attr(class): %s, e.ChildAttrs(a[href], href): %s\n", e.Attr("class"), e.ChildAttrs("a[href]", "href"))
		cAttrVal := e.Attr("class")
		link := e.ChildAttr("a[href]", "href")
		//fmt.Printf("e.Attr(class): %s, e.ChildAttr(a[href], href): %s\n", cAttrVal, link)
		if cAttrVal == "pagination" && strings.Index(link, "http") != -1 {
			//fmt.Println("Trigger Visiting Pagination = ", link)
			e.Request.Visit(link)
		}
	})

	/***
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		// Print link
		fmt.Printf("Link found: %q -> %s\n", e.Text, link)
	})
	***/

	c.OnScraped(func(r *colly.Response) {
		if r.StatusCode != 200 {
			fmt.Println("OnScraped, StatusCode = ", r.StatusCode)
		}
		//fmt.Println("OnScraped, Body = ", string(r.Body))
	})

	c.Visit(url)
	return rp
}

func gatherPrestigeBuilder(prp *[]Residence) {
	var startURL string
	var city string
	var excludeList []string
	var propList []string

	startURL = "https://www.prestigeconstructions.com/residential-bangalore-property/"
	city = "Bangalore"
	propList = prestigeResidenceCollector(startURL, excludeList)
	prestigeResidenceDetails(propList, city, prp)
	//prettyJSON, _ = json.MarshalIndent(propList, "", "    ")
	//fmt.Println("Prestige Bangalore Properties")
	//fmt.Println(string(prettyJSON))

	startURL = "https://www.prestigeconstructions.com/residential-apartments-villas-chennai/"
	excludeList = append(excludeList, "falcon-city")
	city = "Chennai"
	propList = prestigeResidenceCollector(startURL, excludeList)
	prestigeResidenceDetails(propList, city, prp)

	startURL = "https://www.prestigeconstructions.com/residential-apartments-villas-kochi/"
	city = "Kochi"
	propList = prestigeResidenceCollector(startURL, excludeList)
	prestigeResidenceDetails(propList, city, prp)
}

func generateCmdLineHandler() {
	var prp []Residence
	var prettyJSON []byte
	gatherPrestigeBuilder(&prp)
	prettyJSON, _ = json.MarshalIndent(prp, "", "    ")
	fmt.Println("Prestige Property Details")
	fmt.Println(string(prettyJSON))
	err := ioutil.WriteFile(dbFile, prettyJSON, 0644)
	if err != nil {
		fmt.Println("Unable to write to file")
	}

}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to Real Estate Propery Listings")
}

func generateHandler(w http.ResponseWriter, r *http.Request) {
	var prettyJSON []byte
	var err error

	if !fileExists(dbFile) {
		fmt.Println("Generating Database")
		generateCmdLineHandler()
	}
	prettyJSON, err = ioutil.ReadFile(dbFile)
	if err != nil {
		fmt.Println("Unable to read from file")
	}
	fmt.Fprintf(w, "Prestige Property Details, %s", string(prettyJSON))
}

func main() {
	serverMode := false

	//fmt.Println("arglen : ", len(os.Args))
	if len(os.Args) == 2 && os.Args[1] == "server" {
		fmt.Println("In Web Server mode")
		serverMode = true
	}

	if serverMode {
		http.HandleFunc("/", homeHandler)
		http.HandleFunc("/generate/", generateHandler)
		http.ListenAndServe(":8080", nil)
	} else {
		generateCmdLineHandler()
	}
}
