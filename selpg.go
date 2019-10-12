package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"math"
	"github.com/spf13/pflag"
)

type selpgArgs struct {
	startPage  int
	endPage    int
	inFilename string
	pageLen    int
	pageType   string
	printDest  string
}

type spArgs selpgArgs

var progname string /* program name, for error messages */

func main() {
	var sa = spArgs{}
	/* save name by which program is invoked, for error messages */
	progname = os.Args[0]

	process_args(&sa)//接受参数
	process_input(sa)//执行命令
}

func process_args(sa *spArgs) {
	//解析参数
	pflag.IntVarP(&(sa.startPage), "startPage", "s", -1, "start page")
	pflag.IntVarP(&(sa.endPage), "endPage", "e", -1, "end page")
	pflag.IntVarP(&(sa.pageLen), "pageLength", "l", 40, "the length of page")
	pflag.StringVarP(&sa.pageType, "type", "f", "l", "'l': lines-delimited, 'f': form-feed-delimited. default is 'l'")
	pflag.Lookup("type").NoOptDefVal = "f"
	pflag.StringVarP(&sa.printDest, "dest", "d", "", "print destination")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "USAGE: \n%s -sstartPage -eendPage [ -f | -l lines_per_page ]"+" [ -d dest ] [ inFilename ]\n")
		pflag.PrintDefaults()
	}
	pflag.Parse()

	/* check the command-line arguments for validity */
	if len(os.Args) < 3 { /* Not enough args, minimum command is "selpg -sstartpage -eendPage"  */
		fmt.Fprintf(os.Stderr, "\n%s: not enough arguments\n", progname)
		pflag.Usage()
		os.Exit(1)
	}

	/* handle 1st arg - start page */
	if os.Args[1] != "-s" {
		fmt.Fprintf(os.Stderr, "\n%s: 1st arg should be -s startPage\n", progname)
		pflag.Usage()
		os.Exit(2)
	}
	if sa.startPage < 1 || sa.startPage > math.MaxInt32 {
		fmt.Fprintf(os.Stderr, "\n%s: invalid start page %s\n", progname, os.Args[2])
		pflag.Usage()
		os.Exit(3)
	}

	/* handle 2nd arg - end page */
	if os.Args[3] != "-e" {
		fmt.Fprintf(os.Stderr, "\n%s: 2nd arg should be -e endPage\n", progname)
		pflag.Usage()
		os.Exit(4)
	}
	if sa.endPage < 1 || sa.endPage > math.MaxInt32 || sa.endPage < sa.startPage {
		fmt.Fprintf(os.Stderr, "\n%s: invalid end page %s\n", progname, sa.endPage)
		pflag.Usage()
		os.Exit(5)
	}

	if len(pflag.Args()) == 1 {
		_, err := os.Stat(pflag.Args()[0])
		/* check if file exists */
		if err != nil && os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "\n%s: input file \"%s\" does not exist\n",
				progname, pflag.Args()[0])
			os.Exit(6)
		}
		sa.inFilename = pflag.Args()[0]
	}
}

//执行命令
func process_input(sa spArgs) {
	var fin *os.File        /* input stream */
	var fout io.WriteCloser

	/* set the input source */
	if len(sa.inFilename) == 0 {
		fin = os.Stdin
	} else {
		var err error
		fin, err = os.Open(sa.inFilename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n%s: could not open input file \"%s\"\n",
				progname, sa.inFilename)
			os.Exit(7)
		}
		defer fin.Close()
	}

	/* set the output destination */
	bufferFin := bufio.NewReader(fin)

	cmd := &exec.Cmd{}
	//决定输出在文件还是命令行
	if sa.printDest == "" { //命令行
		fout = os.Stdout
	} else {
		var err error
		cmd = exec.Command("cat")//文件
		cmd.Stdout, err = os.OpenFile(sa.printDest, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		fout, err = cmd.StdinPipe()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n%s: can't open pipe to \"lp -d%s\"\n", progname, sa.printDest)
			os.Exit(8)
		}
		cmd.Start()
	}

	/* begin one of two main loops based on page type */
	var pageNum int

	if sa.pageType == "l" {
		lineNum := 0
		pageNum = 1
		for {
			line, crc := bufferFin.ReadString('\n')
			if crc != nil {
				break
			}
			lineNum ++
			if lineNum > sa.pageLen {
				pageNum ++
				lineNum = 1
			}

			if (pageNum >= sa.startPage) && (pageNum <= sa.endPage) {
				//输出到命令行
				_, err := fout.Write([]byte(line))
				if err != nil {
					fmt.Println(err)
					os.Exit(9)
				}
			}
		}
	} else {
		pageNum = 1
		for {
			line, crc := bufferFin.ReadString('\n')
			if crc != nil {
				break
			}

			if (pageNum >= sa.startPage) && (pageNum <= sa.endPage) {
				_, err := fout.Write([]byte(line))
				if err != nil {
					os.Exit(5)
				}
			}
			pageNum ++
		}
	}
	cmd.Wait()
	defer fout.Close()
	/* end main loop */
	if pageNum < sa.startPage {
		fmt.Fprintf(os.Stderr, "\n%s: startPage (%d) greater than total pages (%d)," + " no output written\n", progname, sa.startPage, pageNum)
	} else if pageNum < sa.endPage {
		fmt.Fprintf(os.Stderr, "\n%s: endPage (%d) greater than total pages (%d)," + " less output than expected\n", progname, sa.endPage, pageNum)
	}
}
