package cf

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type InfluxdbPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]float64
	Time        time.Time
}

var (
	InfluxdbWriteUrl = "http://10.11.1.33:8086/write?db=statsd"
)

func WritePoints(points []InfluxdbPoint) error {
	lines := make([]string, 0, len(points))
	for _, p := range points {
		tagStrList := make([]string, 0, len(p.Tags))
		for k, v := range p.Tags {
			tagStrList = append(tagStrList, fmt.Sprintf("%s=%s", k, v))
		}
		tagStr := strings.Join(tagStrList, ",")
		fieldStrList := make([]string, 0, len(p.Fields))
		for k, v := range p.Fields {
			fieldStrList = append(fieldStrList, fmt.Sprintf("%s=%f", k, v))
		}
		fieldStr := strings.Join(fieldStrList, ",")
		line := fmt.Sprintf("%s,%s %s %d", p.Measurement, tagStr, fieldStr, p.Time.UnixNano())
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n")

	req, err := http.NewRequest("POST", InfluxdbWriteUrl, bytes.NewBufferString(content))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return err
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return err
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
	return nil
}
