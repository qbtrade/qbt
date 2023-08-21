package cmd

import (
	"encoding/csv"
	"errors"
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

type tcpPingVar struct {
	//cnt记录进行了多少次tcp-ping,lossCnt记录有多少次丢包
	cnt     int
	lossCnt int
	sumRtt  time.Duration
	//使用队列来记录最近100和1000次的rtt
	rtts100  *tcpPingQueue
	rtts1000 *tcpPingQueue
}

type tcpInformation struct {
	start    time.Time
	hostName string
	ip       string
	port     string
	rtt      time.Duration
	loss     bool
}

func newTcpPingVar() *tcpPingVar {
	return &tcpPingVar{
		sumRtt:   time.Duration(0),
		rtts100:  newTcpPingQueue(100),
		rtts1000: newTcpPingQueue(1000),
	}
}

//tpv结构体中包含实现tcp-ping并发需要的全局变量
var tpv = newTcpPingVar()

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

func tcpSummary(timeout time.Duration) {
	//计算总的平均值
	meanRtt := tpv.sumRtt.Milliseconds() / int64(tpv.cnt)
	fmt.Println("\n当前总共进行了", tpv.cnt, "次tcp-ping其中：")
	fmt.Println("有", tpv.lossCnt, "次连接失败", "平均RTT:", meanRtt, "ms")

	//计算最近100次的统计
	loss100 := tpv.rtts100.LossCount(timeout * time.Second)
	stdDev := tpv.rtts100.StdDev()
	fmt.Println("最近100次中", loss100, "次连接失败", "平均RTT为", tpv.rtts100.mean,
		"最大rtt是", tpv.rtts100.max, "标准差为", stdDev)

	//计算最近1000次的统计
	loss1000 := tpv.rtts1000.LossCount(timeout * time.Second)
	stdDev1000 := tpv.rtts1000.StdDev()
	fmt.Println("最近1000次中", loss1000, "次连接失败", "平均RTT为", tpv.rtts1000.mean,
		"最大rtt是", tpv.rtts1000.max, "标准差为", stdDev1000)

	fmt.Println()
}

func writeCSVRow(csvWriteChan chan tcpInformation, writer *csv.Writer, displaySummaryOnly bool,
	timeout time.Duration) {
	for {
		select {
		case t, ok := <-csvWriteChan:
			//管道已关闭但还是进行读取
			if !ok {
				panic("csvWriteChan channel closed")
			}
			//每次拨号cnt都要++
			tpv.cnt++
			if t.loss {
				tpv.lossCnt++
			}
			tpv.sumRtt += t.rtt
			rttMs := float64(t.rtt.Nanoseconds()) / 1e6
			//将当前rtt加入队列
			tpv.rtts100.pushAndMaintain(t.rtt)
			tpv.rtts1000.pushAndMaintain(t.rtt)

			if !displaySummaryOnly {
				fmt.Printf("\rtcp-ping (%s:%s) seq=%d rtt=%.2fms       ", t.ip, t.port, tpv.cnt, rttMs)
			}

			dataRow := []string{
				strconv.FormatInt(t.start.UnixMilli(), 10),
				t.hostName,
				t.ip,
				t.port,
				strconv.FormatFloat(rttMs, 'f', 4, 64),
				strconv.FormatBool(t.loss),
			}

			err := writer.Write(dataRow)
			if err != nil {
				fmt.Println("writer.Write error", err)
				return
			}

			//每进行100次tcp-ping进行一次统计
			if tpv.cnt%100 == 0 {
				tcpSummary(timeout)
			}

		//刷盘
		case <-time.After(time.Second * 10):
			writer.Flush()
			err := writer.Error()
			if err != nil {
				fmt.Println("writer error", err)
				return
			}

		}

	}
}

func establishTcp(ip, port, hostName string, timeout time.Duration,
	tcpChan chan int, csvWrite chan tcpInformation) {
	//从管道中获得一个许可，防止并发的tcp连接过多
	tcpChan <- 9
	defer func() {
		//还给管道一个许可
		<-tcpChan
	}()

	//拨号，建立TCP连接
	start := time.Now()
	conn, err := net.DialTimeout("tcp", ip+":"+port, timeout*time.Second)
	rtt := time.Duration(0)
	loss := false
	if err != nil {
		loss = true
		//判断是否是超时错误
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			rtt = timeout * time.Second
		} else {
			fmt.Println("连接失败")
			rtt = timeout * time.Second * 2
		}

	} else {
		//确定rtt并写入csv文件
		rtt = time.Since(start)
		defer func() {
			err = conn.Close()
			if err != nil {
				fmt.Println("关闭管道失败:", err)
			}
		}()
	}

	tcpInfo := tcpInformation{
		ip:       ip,
		port:     port,
		hostName: hostName,
		loss:     loss,
		rtt:      rtt,
		start:    start,
	}
	csvWrite <- tcpInfo

}

func openCsvFile(filename string) (writer *csv.Writer, file *os.File, err error) {
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		//文件不存在则创建csv文件,并添加标题
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("create csv file error:", err)
			return
		}
		writer = csv.NewWriter(file)
		dataRow := []string{"ts", "hostname", "ip", "port", "rtt", "loss"}
		err = writer.Write(dataRow)
		if err != nil {
			fmt.Println("writer.Write error", err)
			return
		}
		writer.Flush()
	} else {
		//存在则打开即可
		file, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("open csv file error:", err)
			return
		}
		writer = csv.NewWriter(file)
		dataRow := []string{""}
		err = writer.Write(dataRow)
		if err != nil {
			fmt.Println("writer.Write error", err)
			return
		}
		writer.Flush()

	}
	return writer, file, err
}

func CheckTcpPing(address, hostName string, interval float64, timeout time.Duration, count int,
	displaySummaryOnly bool, maxTcpConnect int) {
	//求ip和端口号
	part := strings.Split(address, ":")
	ip := part[0]
	port := part[1]
	currentTime := time.Now()
	//文件名
	filename := hostName + "_" + "tcp_ping" + "_" + currentTime.Format("2006010215") + ".csv"
	//文件相关变量
	var (
		file   *os.File
		err    error
		writer *csv.Writer
	)
	//用于限制同时执行的线程数量的管道
	tcpChan := make(chan int, maxTcpConnect)
	//用于传递给写线程数据的管道
	csvWriteChan := make(chan tcpInformation, 1000)

	//打开文件
	writer, file, err = openCsvFile(filename)
	if err != nil {
		fmt.Println("open csvFile fail")
		return
	}
	//关闭文件和连接以及管道
	defer func() {
		err = file.Close()
		if err != nil {
			fmt.Println("close file error", err)
			return
		}
	}()
	//写线程
	go writeCSVRow(csvWriteChan, writer, displaySummaryOnly, timeout)

	for count > 0 || tpv.cnt <= count {

		//建立tcp连接
		go establishTcp(ip, port, hostName, timeout, tcpChan, csvWriteChan)

		//等待interval秒再进行查询
		time.Sleep(time.Duration(interval*1000) * time.Millisecond)
	}
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
		maxTcpConnect, _ := cmd.Flags().GetInt("maxTcpConnect")
		hostname, _ := os.Hostname()
		CheckTcpPing(address, hostname, interval, time.Duration(timeout), count, onlySummary, maxTcpConnect)
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
	tcpPingCmd.Flags().IntP("maxTcpConnect", "", 1000, "the maximum number of TCP connections")
}
