package main

import (
	"io"
	"user_st"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	// "log"
)


// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	rAndroid := regexp.MustCompile("Android")
	rMSIE := regexp.MustCompile("MSIE")
	seenBrowsers := make(map[string]bool)
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")

	users := make([]user_st.User, 0, len(lines))
	for _, line := range lines {
		user := user_st.User{}
		// fmt.Printf("%v %v\n", err, line)
		err := user.UnmarshalJSON([]byte(line))
		if err != nil {
			panic(err)
		}
		users = append(users, user)
	}

	for i, user := range users {

		isMSIE := false
		isAndroid := false
		browsers := user.Browsers
		if len(browsers) == 0 {
			// log.Println("cant cast browsers")
			continue
		}

		for _, browserRaw := range browsers {
			browser := browserRaw
			isFind := false
			if isFind = rAndroid.MatchString(browser); isFind {
				isAndroid = true
			} else if isFind = rMSIE.MatchString(browser); isFind {
				isMSIE = true
			}
			if isFind {
				if _, ok := seenBrowsers[browser]; ok {
					uniqueBrowsers++
				} else {
					seenBrowsers[browser] = true
				}
			}
		}


		if !(isAndroid && isMSIE) {
			continue
		}

		// log.Println("Android and MSIE user:", user["name"], user["email"])
		email := r.ReplaceAllString(user.Email, " [at] ")
		foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))
}
