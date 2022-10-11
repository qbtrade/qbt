/*
Copyright © 2022 hinachen <hinachen@1token.trade>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/qbtrade/qbt/cmd/qbt/cf"
	"github.com/spf13/cobra"
	"math"
	"net"
	"time"
)

func Marshal(c any) string {
	v, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(v)
}

// monitorTCPCmd get tcp conn
var monitorTCPCmd = &cobra.Command{
	Use:   "monitor-tcp",
	Short: "get tcp connection",
	Long:  `get tcp connection`,
	Args: func(cmd *cobra.Command, args []string) error {
		ls, _ := cmd.Flags().GetStringSlice("addresses")
		if len(ls) == 0 && len(args) == 0 {
			return fmt.Errorf("no address to connect")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cc := new(ConnConfig)
		cc.Timeout, _ = cmd.Flags().GetInt("timeout")
		cc.Interval, _ = cmd.Flags().GetInt("interval")
		cc.Count, _ = cmd.Flags().GetInt("count")
		cc.Addresses, _ = cmd.Flags().GetStringSlice("addresses")
		if len(args) > 0 { // 支持放在其他参数中 e.g.  qbt monitor-tcp -i 10 -c 10 -a 1.2.3.4:80,2.3.4.5:22 3.4.5.6:8000 4.5.6.7:8001
			cc.Addresses = append(cc.Addresses, args...)
		}
		fmt.Println("init args", Marshal(cc))
		cnt := 0
		summary := newStaticsMsg()
		stage := newStaticsMsg()
		for cc.Count > cnt {
			for _, address := range cc.Addresses {
				cnt += 1
				d, err := connectTCP(address, time.Duration(cc.Timeout), cnt)
				if err != nil {
					stage.FailLength += 1
				} else {
					stage.SuccessLength += 1
					stage.SuccessCost = append(stage.SuccessCost, d)
					stage.MaxCost = cf.Max(stage.MaxCost, d)
					stage.MinCost = cf.Min(stage.MinCost, d)
				}
				if cnt%100 == 0 {
					fmt.Printf("stage information: [%s]\n", stage.String())
					mergeStaticMsg(summary, stage)
					fmt.Printf("summary information: [%s]\n", summary.String())
					stage = newStaticsMsg()
				}
				time.Sleep(time.Duration(cc.Interval) * time.Second)
			}
		}
	},
}

func newStaticsMsg() *StaticsMsg {
	return &StaticsMsg{
		SuccessCost: make([]int64, 0, 0),
		MinCost:     math.MaxInt64,
	}
}

type StaticsMsg struct {
	SuccessCost   []int64 // 成功耗时
	SuccessLength int     // 成功的次数
	FailLength    int     // 失败的次数
	MaxCost       int64   // 成功最大耗时
	MinCost       int64   // 成功最少耗时
	MeanCost      int64   // 成功平均耗时
}

// mergeStaticMsg 将100个ping信息合并到总的里
func mergeStaticMsg(s1 *StaticsMsg, s2 *StaticsMsg) {
	s1.SuccessCost = append(s1.SuccessCost, s2.SuccessCost...)
	s1.SuccessLength += s2.SuccessLength
	s1.FailLength += s2.FailLength
	s1.MaxCost = cf.Max(s1.MaxCost, s2.MaxCost)
	s1.MinCost = cf.Min(s1.MinCost, s2.MinCost)
}

func (s *StaticsMsg) String() string {
	s.MeanCost = cf.Mean(s.SuccessCost)
	return fmt.Sprintf("susscess:%d, fail:%d, max cost:%d, min cost:%d, mean cost:%d", s.SuccessLength, s.FailLength, s.MaxCost, s.MinCost, s.MeanCost)
}

type ConnConfig struct {
	Timeout   int      // 超时
	Interval  int      // 连接间隔
	Count     int      // 最大连接次数
	Addresses []string // 要连接的地址
}

//connectTCP 建立TCP连接
func connectTCP(address string, timeout time.Duration, cnt int) (int64, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout*time.Second)
	if err != nil {
		fmt.Println("connect address error", err)
		return 0, err
	}
	duration := time.Since(start) // tcp 连接的时间间隔
	d := duration.Milliseconds()
	fmt.Println(cnt, time.Now().Format(time.RFC3339), "tcp connect cost:", fmt.Sprintf("%dms", d))
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Println("close tcp conn error,", err)
		}
	}()
	return d, nil
}

func init() {
	rootCmd.AddCommand(monitorTCPCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// 添加局部命令行参数
	monitorTCPCmd.Flags().IntP("timeout", "t", 5, "connect timeout")
	monitorTCPCmd.Flags().IntP("interval", "i", 2, "connect interval")
	monitorTCPCmd.Flags().IntP("count", "c", math.MaxInt, "max count try to connect")
	//monitorTCPCmd.Flags().IntP("loop", "l", math.MaxInt, "max count for loop")
	monitorTCPCmd.Flags().StringSliceP("addresses", "a", []string{}, "want to connect addresses slice such as a,b,c")
}
