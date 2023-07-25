package cmd

import (
	"encoding/csv"
	"fmt"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type tcpPingQueue []int64

func (q *tcpPingQueue) push(item int64) {
	*q = append(*q, item)
}
func (q *tcpPingQueue) pop() (int64, bool) {
	if len(*q) == 0 {
		return 0, false
	}
	item := (*q)[0]
	*q = (*q)[1:]
	return item, true
}

func compareRtt(a, b time.Duration) time.Duration {
	if a > b {
		return a
	} else {
		return b
	}
}

func CheckTcpPing(address, hostName string, interval float64) {
	//求ip和端口号
	part := strings.Split(address, ":")
	ip := part[0]
	port := part[1]

	filename := "tcp_ping.csv"
	var (
		dataRow []string
		file    *os.File
		err     error
		writer  *csv.Writer
	)
	//判断csv文件是否存在
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		//文件不存在则创建csv文件,并添加标题
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("create csv file error:", err)
		}
		writer = csv.NewWriter(file)
		dataRow = []string{"welcome tcp-ping"}
		err := writer.Write(dataRow)
		if err != nil {
			fmt.Println("writer.Write error", err)
		}
		writer.Flush()
	} else {
		//存在则打开即可
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("open csv file error:", err)
		}
		writer = csv.NewWriter(file)
	}

	//cnt记录进行了多少次tcp-ping,lossCnt记录有多少次丢包
	cnt, lossCnt, loss100, loss1000 := 0, 0, 0, 0
	//记录总的rtt，最近100次的总rtt，最近1000次的总的rtt
	sumRtt, sumRtt100, sumRtt1000 := time.Duration(0), time.Duration(0), time.Duration(0)
	//记录总的最大rtt，最近100次的最大rtt，最近1000次的最大rtt
	maxRtt, maxRtt100, maxRtt1000 := time.Duration(0), time.Duration(0), time.Duration(0)
	//使用队列来记录最近100和1000次的rtt
	var rtts100, rtts1000 tcpPingQueue
	//循环tcp-ping
	for {
		//每次尝试进行连接cnt都要+1
		cnt++
		//保证队列不越界
		for len(rtts100) >= 100 {
			item, ok := rtts100.pop()
			if !ok {
				fmt.Println("queue pop fail")
				return
			}
			//保证连接失败数据统计正确
			if item == 0 {
				loss100--
			} else {
				sumRtt100 -= time.Duration(item * 1000000)
			}
		}
		for len(rtts1000) >= 1000 {
			item, ok := rtts1000.pop()
			if !ok {
				fmt.Println("queue pop fail")
				return
			}
			//保证连接失败数据统计正确
			if item == 0 {
				loss1000--
			} else {
				sumRtt1000 -= time.Duration(item * 1000000)
			}

		}
		//拨号，建立TCP连接
		start := time.Now()
		now := start.UnixMilli()
		conn, err := net.DialTimeout("tcp", address, 2*time.Second)
		if err != nil {
			//判断是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				lossCnt++
				loss100++
				loss1000++
				dataRow = []string{strconv.FormatInt(now, 10), hostName, ip, port, "9999", "true"}
				rtts1000.push(0)
				rtts100.push(0)
			} else {
				fmt.Println("连接失败")
				break
			}
		} else {
			//确定rtt并写入csv文件
			rtt := time.Since(start)
			dataRow = []string{strconv.FormatInt(now, 10), hostName, ip, port, rtt.String(), "false"}
			//求总的rtt
			sumRtt += rtt
			sumRtt100 += rtt
			sumRtt1000 += rtt
			//求最大rtt
			maxRtt = compareRtt(maxRtt, rtt)
			//将当前rtt加入队列
			rtts100.push(rtt.Milliseconds())
			rtts1000.push(rtt.Milliseconds())

		}
		//每进行100次tcp-ping进行一次统计
		if cnt%100 == 0 {
			//计算总的平均值
			meanRtt := sumRtt.Milliseconds() / int64(cnt)
			fmt.Println("当前总共进行了", cnt, "次tcp-ping其中：")
			fmt.Println("有", lossCnt, "次连接失败", "平均值为", meanRtt, "ms", "最大rtt为", maxRtt)

			//计算平均值
			meanRtt100 := sumRtt100.Milliseconds() / int64(len(rtts100))
			fmt.Println(sumRtt100)
			//计算方差
			varianceRtt100 := 0.0
			maxRtt100 = time.Duration(0)
			for _, value := range rtts100 {
				maxRtt100 = compareRtt(maxRtt100, time.Duration(value*1000000))
				varianceRtt100 += math.Pow(float64(value-meanRtt100), 2)
			}
			varianceRtt100 /= float64(len(rtts100))
			//计算标准差
			stdDevRtt := math.Sqrt(varianceRtt100)
			fmt.Println("最近100次中", loss100, "次连接超时", "平均RTT为", meanRtt100, "ms", "最大rtt是", maxRtt100, "标准差为", stdDevRtt)

			if cnt%1000 == 0 {
				//计算平均值
				meanRtt1000 := sumRtt1000.Milliseconds() / int64(len(rtts1000))
				//计算方差
				varianceRtt1000 := 0.0
				maxRtt1000 = time.Duration(0)
				for _, value := range rtts1000 {
					maxRtt1000 = compareRtt(maxRtt1000, time.Duration(value*1000000))
					varianceRtt1000 += math.Pow(float64(value-meanRtt1000), 2)
				}
				varianceRtt1000 /= float64(len(rtts1000))
				//计算标准差
				stdDevRtt := math.Sqrt(varianceRtt1000)
				fmt.Println("最近1000次中", loss1000, "次连接超时", "平均RTT为", meanRtt1000, "ms", "最大rtt是", maxRtt1000, "标准差为", stdDevRtt)

			}
			fmt.Println()
		}
		//写入写缓冲区并写入文件
		err = writer.Write(dataRow)
		if err != nil {
			fmt.Println("writer.Write error", err)
		}
		writer.Flush()

		//判断写缓冲区是否有错误
		err = writer.Error()
		if err != nil {
			fmt.Println("writer error", err)
		}

		err = conn.Close()
		if err != nil {
			fmt.Println("close tcp_ping conn error,", err)
		}
		//等待interval秒再进行查询
		time.Sleep(time.Duration(interval*1000) * time.Millisecond)
	}
	//关闭文件和连接
	defer func() {
		err = file.Close()
		if err != nil {
			fmt.Println("close file error", err)
		}
	}()

}
