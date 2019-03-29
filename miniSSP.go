package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/rs/xid"
)

// 外のパッケージからの参照を有効にするためには頭を大文字にする！
type SSPRes struct {
	Url string
}

type DSPRequest struct {
	Ssp_name     string
	Request_time string
	Request_id   string
	App_id       int
}

type WinNotice struct {
	Request_id string
	Price      int
}

// DSPにリクエストを送る
func SendRequest(url string, sendData interface{}) (*http.Response, error) {

	send, err := json.Marshal(sendData)
	if err != nil {
		return nil, err
	}

	sendreq, err := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer([]byte(send)),
	)
	if err != nil {
		return nil, err
	}

	// Content-Type 設定
	sendreq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(sendreq)
	if err != nil {
		return nil, err
	}

	return response, err

}

// WinNoticeを勝者に送るメソッド
func PostWinNotice(url, request_id string, price int) error {

	// WinNoticeのJSONデータ
	winNoticeJson := WinNotice{request_id, price}

	response, err := SendRequest(url, winNoticeJson)

	fmt.Println(response)

	return err

}

// DSPに対してリクエストを送り、レスポンスを受け取り返り値で返す
func ComminucationDSP(url string, dspRequestJson interface{}) (string, string, int, error) {

	response, err := SendRequest(url, dspRequestJson)
	if err != nil {

		return "", "", 0, err
	}

	// レスポンスを受け取る
	length, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		return "", "", 0, err
	}

	body := make([]byte, length)
	length, err = response.Body.Read(body)
	if err != nil && err != io.EOF {
		return "", "", 0, err
	}

	var dspresponse map[string]interface{}
	err = json.Unmarshal(body[:length], &dspresponse)
	if err != nil {
		return "", "", 0, err
	}

	defer response.Body.Close()

	return dspresponse["request_id"].(string), dspresponse["url"].(string), int(dspresponse["price"].(float64)), err

}

// DSPに対してリクエストを送り、レスポンスを受け取り返り値で返す
func PostRequest(url, ssp_name, request_id string, app_id int) (string, string, int, error) {

	// 現在時刻の計算
	nowtime := time.Now()
	const layout = "20060102-150405.0000"
	reqest_time := nowtime.Format(layout)

	// DSPRequestのJSONデータ（DSPに対してリクエストを送る）
	dspRequestJson := DSPRequest{ssp_name, reqest_time, request_id, app_id}

	Request_id := ""
	Url := ""
	Price := 0
	err := errors.New("")
	fmt.Println(err) // Warningをなくすため

	ch := make(chan string)
	for i := 0; i < 1; i++ {
		go func() {
			// DSPに通信して、リクエストを得る
			Request_id, Url, Price, err = ComminucationDSP(url, dspRequestJson)
			ch <- "ok"
		}()
	}

	// タイムアウトを設定する
	timeout := time.After(100 * time.Millisecond)

	for i := 0; i < 1; i++ {
		select {
		case result := <-ch:
			fmt.Println(result)
		case <-timeout:
			err := errors.New("timeout")
			return "", "", 0, err
		}
	}

	return Request_id, Url, Price, nil

}

// SSP本体
func SSPHandle(w http.ResponseWriter, req *http.Request) {

	// リクエストを受け取る
	if req.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	length, err := strconv.Atoi(req.Header.Get("Content-Length"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body := make([]byte, length)
	length, err = req.Body.Read(body)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var appid map[string]interface{}
	err = json.Unmarshal(body[:length], &appid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println(reflect.TypeOf(appid))

	// ssp_nameを設定する
	ssp_name := "Sawa-minissp"

	// UUIDを求める
	uuid := xid.New().String()

	// request_idを設定する
	request_id := ssp_name + "-" + uuid

	// 1stPriceを格納する変数
	first_Price := 1
	// 2ndPrice（入札金額）を格納する変数
	second_Price := 1

	// 1stPriceの広告URLを格納する変数
	// 初期値は自社広告のURL
	tender_Advertisement_Url := "自社広告のURLです！"

	// 1stPriceのrequest_idを格納する変数
	tender_request_id := ""

	// 複数のDSPに対してRequestをPOST、レスポンスを受け取る
	// リクエストを送るDSPの個数
	const request_count = 3

	// ここでfor文か並列処理
	url := []string{"http://10.100.100.20/req", "http://10.100.100.22/req", "http://10.100.100.24/req"}

	// app_idをintに変換
	app_id := int(appid["app_id"].(float64))

	// DSPからのレスポンスの値を代入する配列
	var all_request_id [request_count]string
	var all_url [request_count]string
	var all_price [request_count]int

	// 並列処理をする
	var wg sync.WaitGroup
	for i := 0; i < request_count; i++ {
		wg.Add(1)
		go func(j int) {
			all_request_id[j], all_url[j], all_price[j], err = PostRequest(url[j], ssp_name, request_id, app_id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer wg.Done()
		}(i)
	}

	wg.Wait()

	// どのリクエストが金額が高いか計算
	for i := 0; i < request_count; i++ {

		if first_Price < all_price[i] {
			second_Price = first_Price
			first_Price = all_price[i]
			tender_Advertisement_Url = all_url[i]
			tender_request_id = all_request_id[i]
		} else if first_Price == all_price[i] {
			second_Price = first_Price
		} else if second_Price < all_price[i] {
			second_Price = all_price[i]
		}

	}

	// 2ndPrice（入札金額）をつけて、WinNoticeを返す(時間があったら)
	// PostWinNotice(tender_Advertisement_Url, tender_request_id, second_Price)

	// レスポンスが1つだった時の処理（多分いらないからやらない）

	// 入札された広告のURLを返す
	sspResponse := SSPRes{tender_Advertisement_Url}

	res, err := json.Marshal(sspResponse)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	w.Write(res)
	io.WriteString(w, "\n")

}

func main() {
	http.HandleFunc("/ssp", SSPHandle)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
