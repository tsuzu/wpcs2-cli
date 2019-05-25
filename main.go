package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/k0kubun/pp"
)

var (
	client        http.Client
	id, pass, cid string
)

func init() {
	id = os.Getenv("WPCS2_ID")
	pass = os.Getenv("WPCS2_PASS")
}

func getLoginPage() (string, error) {
	req, err := http.NewRequest("GET", "https://wpcs2.herokuapp.com/users/sign_in", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	reg := regexp.MustCompile(`name="authenticity_token" value="([a-zA-Z0-9\+-=/]*)"`)

	token := reg.FindStringSubmatch(string(b))[1]

	return token, nil
}

func getProblemPage(cid, pid int) (string, error) {
	req, err := http.NewRequest("GET", "https://wpcs2.herokuapp.com/contests/"+strconv.Itoa(cid)+"/problems/"+strconv.Itoa(pid), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	reg := regexp.MustCompile(`name="csrf-token" content="([a-zA-Z0-9\+-=/]*)"`)

	token := reg.FindStringSubmatch(string(b))[1]

	return token, nil
}

func login(email, password string) error {
	token, err := getLoginPage()

	if err != nil {
		return err
	}

	token = url.QueryEscape(token)

	email = url.QueryEscape(email)
	password = url.QueryEscape(password)

	body := strings.NewReader(`utf8=%E2%9C%93&authenticity_token=` + token + `&user%5Bemail%5D=` + email + `&user%5Bpassword%5D=` + password + `&user%5Bremember_me%5D=0&commit=%E3%83%AD%E3%82%B0%E3%82%A4%E3%83%B3`)
	req, err := http.NewRequest("POST", "https://wpcs2.herokuapp.com/users/sign_in", body)
	if err != nil {
		return err
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", "https://wpcs2.herokuapp.com")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/users/sign_in")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

type Contest struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Baseline      float64   `json:"baseline"`
	CurrentUserID int       `json:"current_user_id"`
	Joined        bool      `json:"joined"`
	ContestStatus string    `json:"contest_status"`
	AdminRole     bool      `json:"admin_role"`
	Problems      []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		DataSets    []struct {
			ID       int    `json:"id"`
			Label    string `json:"label"`
			MaxScore int    `json:"max_score"`
			Correct  bool   `json:"correct"`
			Score    int    `json:"score"`
		} `json:"data_sets"`
	} `json:"problems"`
	Editorial struct {
		ID        int       `json:"id"`
		ContestID int       `json:"contest_id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"editorial"`
}

func getContest(cidRaw int) (*Contest, error) {
	cid := strconv.Itoa(cidRaw)

	req, err := http.NewRequest("GET", "https://wpcs2.herokuapp.com/api/contests/"+cid, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/contests/"+cid)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cont Contest

	if err := json.NewDecoder(resp.Body).Decode(&cont); err != nil {
		return nil, err
	}

	return &cont, err
}

func getTestCase(cid, pid, tid int) (io.ReadCloser, error) {
	// https://wpcs2.herokuapp.com/contests/7/problems/37/data_sets/56

	req, err := http.NewRequest("GET", "https://wpcs2.herokuapp.com/contests/"+strconv.Itoa(cid)+"/problems/"+strconv.Itoa(pid)+"/data_sets/"+strconv.Itoa(tid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/contests/"+strconv.Itoa(cid)+"/problems/"+strconv.Itoa(pid))
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp.Body, err
}

type SubmitResult struct {
	ID          int       `json:"id"`
	ProblemID   int       `json:"problem_id"`
	DataSetID   int       `json:"data_set_id"`
	JudgeStatus int       `json:"judge_status"`
	Score       int       `json:"score"`
	CreatedAt   time.Time `json:"created_at"`
}

func submitImpl(cid, pid, tid int, reader io.Reader) (*SubmitResult, error) {
	buf := bytes.NewBuffer(nil)

	writer := multipart.NewWriter(buf)

	token, err := getProblemPage(cid, pid)

	if err != nil {
		return nil, err
	}

	writer.WriteField("authenticity_token", token)
	writer.WriteField("data_set_id", strconv.Itoa(tid))
	field, err := writer.CreateFormField("answer")

	if err != nil {
		return nil, err
	}

	io.Copy(field, reader)
	writer.Close()

	req, err := http.NewRequest("POST", "https://wpcs2.herokuapp.com/api/contests/"+strconv.Itoa(cid)+"/submissions", buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Origin", "https://wpcs2.herokuapp.com")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.157 Safari/537.36")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
	req.Header.Set("Referer", "https://wpcs2.herokuapp.com/contests/"+strconv.Itoa(cid)+"/problems/"+strconv.Itoa(pid))
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Origin", "https://wpcs2.herokuapp.com")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var res SubmitResult

	json.NewDecoder(resp.Body).Decode(&res)

	return &res, nil
}

func submit(cid, pidx int, tcname string, reader io.Reader) (*SubmitResult, error) {
	tcname = strings.ToLower(tcname)

	cont, err := getContest(cid)

	if err != nil {
		return nil, err
	}

	prob := cont.Problems[pidx]

	target := -1
	for i := range prob.DataSets {
		if tcname == strings.ToLower(prob.DataSets[i].Label) {
			target = i
			break
		}
	}

	return submitImpl(cid, prob.ID, prob.DataSets[target].ID, reader)
}

const usage = `Set $WPCS2_ID, $WPCS2_PASS, $WPCS2_CID
wpcs2 get pid(A-Z) [small/medium/large]
wpcs2 submit pid(A-Z) [small/medium/large]
wpcs2 server # start server`

var cache sync.Map

func get(cid, pidx int, tcname string) (io.ReadCloser, error) {
	v, ok := cache.Load(fmt.Sprintf("%d-%d-%s", cid, pidx, tcname))

	if ok {
		return ioutil.NopCloser(bytes.NewReader(v.([]byte))), nil
	}

	cont, err := getContest(cid)

	if err != nil {
		return nil, err
	}

	tcname = strings.ToLower(tcname)

	prob := cont.Problems[pidx]

	target := -1
	for i := range prob.DataSets {
		if tcname == strings.ToLower(prob.DataSets[i].Label) {
			target = i
			break
		}
	}

	rc, err := getTestCase(cid, prob.ID, prob.DataSets[target].ID)

	if err != nil {
		return nil, err
	}

	defer rc.Close()

	b, err := ioutil.ReadAll(rc)

	if err != nil {
		return nil, err
	}

	cache.Store(fmt.Sprintf("%d-%d-%s", cid, pidx, tcname), b)

	return ioutil.NopCloser(bytes.NewReader(b)), nil
}

func server() {
	if err := login(id, pass); err != nil {
		panic(err)
	}

	router := gin.New()

	router.GET("/get", func(ctx *gin.Context) {
		cid, _ := strconv.Atoi(ctx.Query("cid"))
		pidx, _ := strconv.Atoi(ctx.Query("pidx"))
		tcname := ctx.Query("tcname")

		res, err := get(cid, pidx, tcname)

		if err != nil {
			ctx.String(http.StatusInternalServerError, "text/html", err.Error())

			return
		}
		io.Copy(ctx.Writer, res)
		res.Close()
	})

	router.POST("/submit", func(ctx *gin.Context) {
		cid, _ := strconv.Atoi(ctx.PostForm("cid"))
		pidx, _ := strconv.Atoi(ctx.PostForm("pidx"))
		tcname := ctx.PostForm("tcname")
		data := ctx.PostForm("body")

		res, err := submit(cid, pidx, tcname, strings.NewReader(data))

		if err != nil {
			ctx.String(http.StatusInternalServerError, "text/plain", err.Error())

			return
		}

		ctx.JSON(http.StatusOK, res)
	})

	router.Run("127.0.0.1:14716")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)

		return
	}

	client = http.Client{}

	jar, err := cookiejar.New(nil)

	if err != nil {
		panic(err)
	}

	client.Jar = jar

	command := os.Args[1]
	os.Args = os.Args[2:]
	switch strings.ToLower(command) {
	case "get":
		v := strings.ToLower(os.Args[0])
		v = strconv.Itoa(int(v[0] - 'a'))

		query := fmt.Sprintf("?cid=%s&pidx=%s&tcname=%s", url.QueryEscape(os.Getenv("WPCS2_CID")), url.QueryEscape(v), url.QueryEscape(os.Args[1]))

		resp, err := http.DefaultClient.Get("http://localhost:14716/get" + query)

		if err != nil {
			panic(err)
		}

		io.Copy(os.Stdout, resp.Body)
		resp.Body.Close()
	case "submit":
		val := url.Values{}

		raw := strings.ToLower(os.Args[0])
		pidx := int(raw[0] - 'a')

		val.Add("cid", os.Getenv("WPCS2_CID"))
		val.Add("pidx", strconv.Itoa(pidx))
		val.Add("tcname", os.Args[1])

		b, _ := ioutil.ReadAll(os.Stdin)
		val.Add("body", string(b))

		resp, err := http.DefaultClient.Post("http://localhost:14716/submit", "application/x-www-form-urlencoded", strings.NewReader(val.Encode()))

		if err != nil {
			panic(err)
		}

		var res SubmitResult
		json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()

		pp.Println(res)
	case "server":
		server()
	default:
		fmt.Println("unknown command")
	}
}
