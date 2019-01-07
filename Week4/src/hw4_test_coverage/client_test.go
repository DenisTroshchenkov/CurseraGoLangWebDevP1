package main

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func checkErrorResp(gotErr error, expErr string, result *SearchResponse, t *testing.T, addMsg string) {
	if gotErr == nil {
		t.Errorf("[%s] expected error: %s", addMsg, expErr)
	}
	if gotErr != nil && !strings.Contains(gotErr.Error(), expErr) {
		t.Errorf("[%s] wrong error, expected %s, got %#v", addMsg, expErr, gotErr)
	}
	if result != nil {
		t.Errorf("[%s] wrong result, expected %#v, got %#v", addMsg, nil, result)
	}
}

func TimeoutServer(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Second * 2)
}

func TestFindUsersTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(TimeoutServer))
	c := &SearchClient{
		URL: ts.URL,
	}
	result, err := c.FindUsers(SearchRequest{})
	checkErrorResp(err, "timeout", result, t, "")
	ts.Close()
}

func InternalServerErrorServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func TestFindUsersInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(InternalServerErrorServer))
	c := &SearchClient{
		URL: ts.URL,
	}
	result, err := c.FindUsers(SearchRequest{})
	checkErrorResp(err, "SearchServer fatal error", result, t, "")
	ts.Close()
}

func BadJsonServer(w http.ResponseWriter, r *http.Request) {
	order_field := r.FormValue("order_field")
	if order_field != "BadRequest" {
		w.WriteHeader(http.StatusBadRequest)
	}
	io.WriteString(w, `{"status": 400`) //broken json
}

func TestFindUsersBadJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(BadJsonServer))
	c := &SearchClient{
		URL: ts.URL,
	}
	for _, item := range []string{"GoodRequest", "BadRequest"} {
		result, err := c.FindUsers(SearchRequest{OrderField: item})
		checkErrorResp(err, "cant unpack", result, t, item)
	}
	ts.Close()
}

const (
	AccessToken = "a41gbfg123sxvcvsedsd12asdxcvx"
)

type XMLRoot struct {
	XMLName xml.Name `xml:"root"`
	Row     []XMLRow `xml:"row"`
}

type XMLRow struct {
	XMLName   xml.Name `xml:"row"`
	Id        int      `xml:"id"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int      `xml:"age"`
	About     string   `xml:"about"`
	Gender    string   `xml:"gender"`
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != AccessToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	order_field := r.FormValue("order_field")
	if order_field != "Id" && order_field != "Age" && order_field != "Name" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400, "error": "ErrorBadOrderField"}`)
		return
	}
	limit, _ := strconv.Atoi(r.FormValue("limit"))
	if limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400, "error": "ErrorLimit"}`)
		return
	}
	offset, _ := strconv.Atoi(r.FormValue("offset"))
	if offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400, "error": "ErrorOffsetStr"}`)
		return
	}
	orderBy, _ := strconv.Atoi(r.FormValue("order_by"))
	if orderBy < -1 || orderBy > 1 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400, "error": "ErrorOrderBy"}`)
		return
	}
	query := r.FormValue("query")
	file, _ := os.Open("dataset.xml")
	defer file.Close()
	fileData, _ := ioutil.ReadAll(file)
	xmlData := &XMLRoot{}
	xml.Unmarshal(fileData, xmlData)
	data := []User{}
	count := 0
	for _, row := range xmlData.Row {
		name := row.FirstName + " " + row.LastName
		if !strings.Contains(row.About, query) && !strings.Contains(name, query) {
			continue
		}
		count++
		if count > limit {
			break
		}
		user := User{
			row.Id,
			name,
			row.Age,
			row.About,
			row.Gender,
		}
		data = append(data, user)
	}
	result, _ := json.Marshal(data)
	w.Write(result)
}

func TestFindUsersUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	c := &SearchClient{
		AccessToken: AccessToken,
	}
	result, err := c.FindUsers(SearchRequest{})
	checkErrorResp(err, "unknown error", result, t, "")
	ts.Close()
}

func TestFindUsersBadAccessToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	c := &SearchClient{
		"xcv213easfsdgf",
		ts.URL,
	}
	result, err := c.FindUsers(SearchRequest{})
	checkErrorResp(err, "Bad AccessToken", result, t, "")
	ts.Close()
}

type TestCase struct {
	Req     SearchRequest
	IsError bool
	Result  *SearchResponse
	Err     string
}

func TestFindUsers(t *testing.T) {
	cases := []TestCase{
		TestCase{
			Req: SearchRequest{
				Limit: -1,
			},
			Result:  nil,
			IsError: true,
			Err:     "limit must be > 0",
		},
		TestCase{
			Req: SearchRequest{
				Limit:  1,
				Offset: -1,
			},
			Result:  nil,
			IsError: true,
			Err:     "offset must be > 0",
		},
		TestCase{
			Req: SearchRequest{
				OrderField: "BadOrderFiled",
			},
			Result:  nil,
			IsError: true,
			Err:     "OrderFeld",
		},
		TestCase{
			Req: SearchRequest{
				OrderField: "Id",
				OrderBy:    -2,
			},
			Result:  nil,
			IsError: true,
			Err:     "unknown bad request error",
		},
		TestCase{
			Req: SearchRequest{
				Query:      "Magna",
				OrderField: "Id",
				Limit:      1,
			},
			Result: &SearchResponse{
				Users: []User{
					User{Id: 20,
						Name:   "Lowery York",
						Age:    27,
						About:  "Dolor enim sit id dolore enim sint nostrud deserunt. Occaecat minim enim veniam proident mollit Lorem irure ex. Adipisicing pariatur adipisicing aliqua amet proident velit. Magna commodo culpa sit id.\n",
						Gender: "male"},
				},
				NextPage: true,
			},
			IsError: false,
		},
		TestCase{
			Req: SearchRequest{
				Query:      "Magna",
				OrderField: "Id",
				Limit:      26,
			},
			Result: &SearchResponse{
				Users: []User{
					User{Id: 20,
						Name:   "Lowery York",
						Age:    27,
						About:  "Dolor enim sit id dolore enim sint nostrud deserunt. Occaecat minim enim veniam proident mollit Lorem irure ex. Adipisicing pariatur adipisicing aliqua amet proident velit. Magna commodo culpa sit id.\n",
						Gender: "male"},
					User{Id: 25,
						Name:   "Katheryn Jacobs",
						Age:    32,
						About:  "Magna excepteur anim amet id consequat tempor dolor sunt id enim ipsum ea est ex. In do ea sint qui in minim mollit anim est et minim dolore velit laborum. Officia commodo duis ut proident laboris fugiat commodo do ex duis consequat exercitation. Ad et excepteur ex ea exercitation id fugiat exercitation amet proident adipisicing laboris id deserunt. Commodo proident laborum elit ex aliqua labore culpa ullamco occaecat voluptate voluptate laboris deserunt magna.\n",
						Gender: "female"},
					User{Id: 34,
						Name:   "Kane Sharp",
						Age:    34,
						About:  "Lorem proident sint minim anim commodo cillum. Eiusmod velit culpa commodo anim consectetur consectetur sint sint labore. Mollit consequat consectetur magna nulla veniam commodo eu ut et. Ut adipisicing qui ex consectetur officia sint ut fugiat ex velit cupidatat fugiat nisi non. Dolor minim mollit aliquip veniam nostrud. Magna eu aliqua Lorem aliquip.\n",
						Gender: "male"},
				},

				NextPage: false,
			},
			IsError: false,
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	for caseNum, item := range cases {
		c := &SearchClient{
			AccessToken,
			ts.URL,
		}
		result, err := c.FindUsers(item.Req)

		if err != nil && !item.IsError {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if err != nil && item.IsError && !strings.Contains(err.Error(), item.Err) {
			t.Errorf("[%d] wrong error, expected %#v, got %#v", caseNum, item.Err, err)
		}
		if err == nil && item.IsError {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}
