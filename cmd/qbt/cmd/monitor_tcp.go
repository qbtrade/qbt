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
	"github.com/spf13/cobra"
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
		if len(args) != 1 {
			return fmt.Errorf("need ip:port")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cc := new(ConnConfig)
		cc.Timeout, _ = cmd.Flags().GetInt("timeout")
		cc.Interval, _ = cmd.Flags().GetInt("interval")
		cc.Count, _ = cmd.Flags().GetInt("count")
		cc.Address = args[0]
		fmt.Println("init args", Marshal(cc))
		for cc.Count > 0 {
			err := connectTCP(cc)
			if err != nil {
				fmt.Println(err)
			}
			//fmt.Println(cc)
			time.Sleep(time.Duration(cc.Interval) * time.Second)
			cc.Count -= 1
			//fmt.Println(cc.Count)
		}
		return
	},
}

type ConnConfig struct {
	Timeout  int    // 超时
	Interval int    // 链接间隔
	Count    int    // 最大链接次数
	Address  string // 要链接的地址
}

// connectTCP 建立TCP链接
func connectTCP(cc *ConnConfig) error {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", cc.Address, time.Duration(cc.Timeout)*time.Second)
	if err != nil {
		fmt.Println("connect address error", err)
		return err
	}
	duration := time.Since(start) // tcp 链接的时间间隔
	fmt.Println("tcp connect spend:", duration.String())
	defer func() {
		err = conn.Close()
		if err != nil {
			fmt.Println("close tcp conn error,", err)
		}
	}()
	return nil
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
	monitorTCPCmd.Flags().IntP("timeout", "t", 5, "超时时常")
	monitorTCPCmd.Flags().IntP("interval", "i", 10, "链接间隔")
	monitorTCPCmd.Flags().IntP("count", "c", 5, "尝试链接次数")
}
