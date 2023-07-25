package cmd

import (
	"encoding/csv"
	"fmt"
	"github.com/spf13/cobra"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type tcpPingQueue struct {
	limit int
	items []time.Duration
	sum   time.Duration
	mean  time.Duration
	max   time.Duration
}

func newTcpPingQueue(limit int) *tcpPingQueue {
	return &tcpPingQueue{
		limit: limit,
		items: make([]time.Duration, 0, limit),
	}
}

func (q *tcpPingQueue) pushAndMaintain(item time.Duration) (first time.Duration, full bool) {
	q.items = append(q.items, item)
	q.sum += item
	if len(q.items) > q.limit {
		first = q.items[0]
		q.items = q.items[1:]
		q.sum -= first
		full = true
	}
	q.mean = q.sum / time.Duration(len(q.items))
	if item > q.max {
		q.max = item
	} else {
		if first >= q.max {
			q.max = 0
			for _, value := range q.items {
				if value > q.max {
					q.max = value
				}
			}
		}
	}
	return first, full
}

func (q *tcpPingQueue) StdDev() time.Duration {
	variance := 0.0
	for _, value := range q.items {
		variance += math.Pow(float64(value-q.mean), 2)
	}
	variance /= float64(len(q.items))
	//计算标准差
	return time.Duration(math.Sqrt(variance))
}

func (q *tcpPingQueue) LossCount(timeout time.Duration) (lossCount int) {
	for _, value := range q.items {
		if value > timeout {
			lossCount++
		}
	}
	return lossCount
}

func compareRtt(a, b time.Duration) time.Duration {
	if a > b {
		return a
	} else {
		return b
	}
}

func CheckTcpPing(address, hostName string, interval float64, timeout time.Duration, count int, displaySummaryOnly bool) {
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
		dataRow = []string{"ts", "hostname", "ip", "port", "rtt", "loss"}
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
	cnt, lossCnt := 0, 0
	sumRtt := time.Duration(0)
	//使用队列来记录最近100和1000次的rtt
	var rtts100 = newTcpPingQueue(100)
	var rtts1000 = newTcpPingQueue(1000)
	//循环tcp-ping
	for {
		//每次尝试进行连接cnt都要+1
		cnt++
		if count > 0 && cnt > count {
			break
		}
		//拨号，建立TCP连接
		start := time.Now()
		now := start.UnixMilli()
		conn, err := net.DialTimeout("tcp", address, timeout*time.Second)
		rtt := time.Duration(0)
		if err != nil {
			//判断是否是超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				lossCnt++
				dataRow = []string{strconv.FormatInt(now, 10), hostName, ip, port, "9999", "true"}
				rtt = timeout * time.Second
			} else {
				fmt.Println("连接失败")
				break
			}
		} else {
			//确定rtt并写入csv文件
			rtt = time.Since(start)
		}
		//将当前rtt加入队列
		sumRtt += rtt
		rtts100.pushAndMaintain(rtt)
		rtts1000.pushAndMaintain(rtt)

		rttMs := float64(rtt.Nanoseconds()) / 1e6
		if !displaySummaryOnly {
			fmt.Printf("\rtcp-ping (%s:%s) seq=%d rtt=%.2fms       ", ip, port, cnt, rttMs)
		}
		dataRow = []string{strconv.FormatInt(now, 10), hostName, ip, port, fmt.Sprintf("%.2f", rttMs), "false"}

		//每进行100次tcp-ping进行一次统计
		if cnt%100 == 0 {
			//计算总的平均值
			meanRtt := sumRtt.Nanoseconds() / int64(cnt)
			fmt.Println("\n当前总共进行了", cnt, "次tcp-ping其中：")
			fmt.Println("有", lossCnt, "次连接失败", "平均RTT:", meanRtt)

			//计算最近100次的统计
			loss100 := rtts100.LossCount(timeout * time.Second)
			stdDev := rtts100.StdDev()
			fmt.Println("最近100次中", loss100, "次连接超时", "平均RTT为", rtts100.mean,
				"最大rtt是", rtts100.max, "标准差为", stdDev)

			//计算最近1000次的统计
			loss1000 := rtts1000.LossCount(timeout * time.Second)
			stdDev1000 := rtts1000.StdDev()
			fmt.Println("最近1000次中", loss1000, "次连接超时", "平均RTT为", rtts1000.mean,
				"最大rtt是", rtts1000.max, "标准差为", stdDev1000)

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

var tcpPingCmd = &cobra.Command{
	Use:   "tcp-ping",
	Short: "ping tcp rtt",
	Long:  `ping tcp rtt`,
	Args: func(cmd *cobra.Command, args []string) error {
		addr, err := cmd.Flags().GetString("address")
		if err != nil || addr == "" {
			return fmt.Errorf("no address to connect")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		onlySummary, _ := cmd.Flags().GetBool("only-summary")
		timeout, _ := cmd.Flags().GetInt("timeout")
		interval, _ := cmd.Flags().GetFloat64("interval")
		count, _ := cmd.Flags().GetInt("count")
		address, _ := cmd.Flags().GetString("address")
		hostname, _ := os.Hostname()
		CheckTcpPing(address, hostname, interval, time.Duration(timeout), count, onlySummary)
	},
}

func init() {
	rootCmd.AddCommand(tcpPingCmd)
	// 添加局部命令行参数
	tcpPingCmd.Flags().BoolP("only-summary", "", false, "display only summary")
	tcpPingCmd.Flags().IntP("timeout", "t", 2, "connect timeout")
	tcpPingCmd.Flags().Float64P("interval", "i", 1, "connect interval")
	tcpPingCmd.Flags().IntP("count", "c", math.MaxInt, "max count try to connect")
	tcpPingCmd.Flags().StringP("address", "a", "10.11.0.1:80", "want to connect to IP:PORT")
}
