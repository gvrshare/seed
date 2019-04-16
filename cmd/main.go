package main

import (
	"fmt"
	"github.com/girlvr/seed"
	"github.com/girlvr/seed/model"
	"github.com/godcong/go-trait"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var rootCmd = &cobra.Command{
	Use:        "seed",
	Aliases:    nil,
	SuggestFor: nil,
	Short:      "Seed is a ipfs video split upload tool",
	Long: `Seed make you split and upload video file with hls in one step
				Complete documentation is available at https://github.com/girlvr/seed`,
}

func main() {
	defer func() {
		if e := recover(); e != nil {
			log.Panic(e)
		}
	}()
	path := rootCmd.PersistentFlags().StringP("path", "p", "seed.json", "set the path to load video source")
	shell := rootCmd.PersistentFlags().StringP("shell", "s", "localhost:5001", "set the ipfs api port")
	//action := rootCmd.PersistentFlags().StringP("action", "a", "cmdProcess", "set action to do something")

	trait.InitRotateLog("logs/seed.log", trait.RotateLogLevel(trait.RotateLogDebug))

	var cmdProcess = &cobra.Command{
		Use:   "process",
		Short: "process split and upload to ipfs",
		Long: `process split a video file to hls ,then upload to ipfs. 
		after that return a ipfs hash info json.`,
		//Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			seed.InitShell(*shell)
			e := model.InitDB()
			if e != nil {
				panic(e)
			}
			vs := seed.Load(*path)
			for _, v := range vs {
				e := seed.Upload(v)
				if e != nil {
					panic(e)
				}
			}

			log.Infof("%+v", vs[0])
		},
	}

	var cmdUpdate = &cobra.Command{
		Use:   "update",
		Short: "update the information",
		Long:  `update only update the video information to new information`,
		//Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(args)
			if len(args) < 1 {
				fmt.Println("nothing process")
				return
			}
			seed.InitShell(*shell)
			e := model.InitDB()
			if e != nil {
				panic(e)
			}
			vs := seed.Load(*path)
			for _, v := range vs {
				e := seed.Update(args[0], v)
				if e != nil {
					panic(e)
				}
			}

			log.Infof("%+v", vs[0])
		},
	}

	var cmdTransfer = &cobra.Command{
		Use:   "transfer",
		Short: "transfer video data info to json",
		Long:  `transfer will output a json file from video info db.`,
		//Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			seed.InitShell(*shell)
			e := model.InitDB()
			if e != nil {
				panic(e)
			}
			if err := seed.Transfer(); err != nil {
				panic(err)
			}
		},
	}
	rootCmd.AddCommand(cmdProcess, cmdTransfer, cmdUpdate)
	rootCmd.SuggestionsMinimumDistance = 1
	Execute()
}

// Execute ...
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
