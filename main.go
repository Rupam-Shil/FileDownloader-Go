package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Dowload struct {
	Url string
	TargetPath string
	TotalSections int  //maximum no of connection
}



func main() {
	fmt.Println("Welcome to concurrency download manager")

	startTime := time.Now()
	// enter file link and destination path here
	d := Dowload{
		Url: "https://www.pexels.com/video/4831675/download/",
		TargetPath: "final.mp4",
		TotalSections : 10,

	}
	err := d.Do()
	errHandle(err)
	fmt.Printf("Dowload complete in %v seconds \n", time.Now().Sub(startTime).Seconds())
}

func (d Dowload) Do() error{
	fmt.Println("Making connection...")
	r, err := d.getNewRequest("HEAD")
	if err != nil{
		return err
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil{
		return err
	}
	fmt.Printf("Got %v \n", resp.StatusCode)
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return fmt.Errorf("can't process, response is %v", resp.StatusCode)
	}
	size, err :=strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil{
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	var section = make([][2]int, d.TotalSections)
	eachSize := size / d.TotalSections
	fmt.Printf("Each size is %v \n", eachSize)
	for i := range section {
		if i ==0 {
			section[i][0] = 0
		}else{
			section[i][0] =  section[i-1][1]+1
		}
		
		if i < d.TotalSections-1{
			section[i][1] = section[i][0] + eachSize
		}else{
			section[i][1] = size - 1
		}
	}
	//Concurrency
	var wg = &sync.WaitGroup{}
	for i, s := range section {
		wg.Add(1)
		i := i
		s := s
		go func(){
			defer wg.Done()
			err := d.downloadSection(i, s)
			if err != nil{
				panic(err)
			}
		}()
	}
	wg.Wait()
	err = d.mergeFiles(section)
	if err != nil {
		return  err
	}
	for i := range section {
		wg.Add(1)
		i:=i
		go func(){
			defer wg.Done()
			err := d.removeTemp(i)
			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()
	return nil;
}

func (d Dowload) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "Download Mangager")
	return r, nil
}

func (d Dowload) downloadSection(i int, s [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range",fmt.Sprintf("bytes=%v-%v",s[0],s[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Downloaded %v bytes for section %v:%v\n", resp.Header.Get("Content-Length"), i, s)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp",i), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (d Dowload) mergeFiles(section [][2]int) error{
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := range section {
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp",i))
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil {
			return err
		}
		fmt.Printf("%v bytes merged\n",n)
	}
	return nil
}
func (d Dowload) removeTemp(i int) error {
	err := os.Remove(fmt.Sprintf("section-%v.tmp",i))
	if err != nil {
		return err
	}
	return nil
}

func errHandle(err error){
	if err != nil {
		log.Fatal(err)
	}
}