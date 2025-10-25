package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	API = "https://api.sourcecraft.tech" 
	client = &http.Client{}  
	PAT = "pv1_QOt95M5kFS43F0WBLl4f43xHUT9zQ6819332l5x76Z61m76Z9d88J93Bm429UxR0_3059427555"
)

func Execute1(method, path string, data map[string] any) (map[string] any, error) {

	if data == nil {
		data = make(map[string] any)
	}

	body, err := ToJson(data)
	Ensure(err)

	return Execute(method, path, body)
}

func Execute(method, path, data string) (map[string] any, error) {
	//println(data)
	if data == "" {
		data = "{}"
	}

	req, err := http.NewRequest(method, API + path, bytes.NewBufferString(data)) 
	Ensure(err)

	req.Header.Set("Accept", "application/json") 
	req.Header.Set("Content-Type", "application/json")  

	if len(PAT) > 0 {
		req.Header.Set("Authorization", "Bearer " + PAT)
	}
	
	resp, err := client.Do(req)  
    Ensure(err) 

    defer resp.Body.Close() 

	if resp.StatusCode != http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("http.error %d", resp.StatusCode)
		}

		return nil, fmt.Errorf("http.error %d, why: %s", resp.StatusCode, string(bytes))
	}
	
	return ReadJson(&resp.Body)
}

func Ensure(err error) {
	if err != nil { panic(err) }  
}

func ReadJson(reader *io.ReadCloser) (map[string] any, error) {
	bytes, err := io.ReadAll(*reader)
    if err != nil {
        return nil, err
    }

    return ParseJson(bytes)
}

func ParseJson(data []byte) (map[string] any, error) {
	var result map[string] any

    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }

    return result, nil
}

func ToJson(m map[string] any) (string, error) {
    b, err := json.MarshalIndent(m, "", "  ")
    
	if err != nil {
        return "", err
    }

    return string(b), nil
}

func HelpIfEmpty() {}