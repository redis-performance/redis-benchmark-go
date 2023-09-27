package main

import (
	"bytes"
	"context"
	"github.com/rueian/rueidis"
	"os"
	"os/exec"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func getTestConnectionDetails() (string, string) {
	value, exists := os.LookupEnv("REDIS_TEST_HOST")
	host := "localhost:6379"
	password := ""
	valuePassword, existsPassword := os.LookupEnv("REDIS_TEST_PASSWORD")
	if exists && value != "" {
		host = value
	}
	if existsPassword && valuePassword != "" {
		password = valuePassword
	}
	return host, password
}

func Test_getplaceholderpos(t *testing.T) {
	type args struct {
		args    []string
		verbose bool
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 int
	}{
		{"empty args", args{[]string{}, false}, -1, -1},
		{"no placeholders", args{[]string{"SET", "k", "v"}, false}, -1, -1},
		{"key placeholder", args{[]string{"SET", "__key__", "v"}, false}, 1, -1},
		{"data placeholder", args{[]string{"SET", "key", "__data__"}, false}, -1, 2},
		{"both placeholders", args{[]string{"SET", "__key__", "__data__"}, false}, 1, 2},
		{"placeholders mixed", args{[]string{"SET", "a(__key__)", "__data__"}, false}, 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKeyPos, gotDataPos := getplaceholderpos(tt.args.args, tt.args.verbose)
			if gotKeyPos != tt.want {
				t.Errorf("getplaceholderpos() got = %v, want %v", gotKeyPos, tt.want)
			}
			if gotDataPos != tt.want1 {
				t.Errorf("getplaceholderpos() got1 = %v, want %v", gotDataPos, tt.want1)
			}
		})
	}
}

func Test_keyBuildLogic(t *testing.T) {
	type args struct {
		keyPos      int
		dataPos     int
		datasize    uint64
		keyspacelen uint64
		cmdS        []string
		charset     string
	}

	tests := []struct {
		name        string
		args        args
		wantNewCmdS []string
		wantKey     string
	}{
		{"no replacement", args{-1, -1, 0, 0, []string{"SET", "key", "value"}, charset}, []string{"SET", "key", "value"}, ""},
		{"key replacement", args{1, -1, 0, 1, []string{"SET", "__key__", "value"}, charset}, []string{"SET", "0", "value"}, "0"},
		{"data replacement", args{1, 2, 0, 1, []string{"SET", "__key__", "__data__"}, "a"}, []string{"SET", "0", ""}, "0"},
		{"data replacement with datasize", args{1, 2, 10, 1, []string{"SET", "__key__", "__data__"}, "a"}, []string{"SET", "0", "aaaaaaaaaa"}, "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewCmdS, gotKey := keyBuildLogic(tt.args.keyPos, tt.args.dataPos, tt.args.datasize, tt.args.keyspacelen, tt.args.cmdS, tt.args.charset)
			if !reflect.DeepEqual(gotNewCmdS, tt.wantNewCmdS) {
				t.Errorf("keyBuildLogic() gotNewCmdS = %v, want %v", gotNewCmdS, tt.wantNewCmdS)
			}
			if gotKey != tt.wantKey {
				t.Errorf("keyBuildLogic() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
		})
	}
}

func TestGecko(t *testing.T) {
	var tests = []struct {
		name               string
		wantExitCode       int
		wantKeyspaceLength int
		wantIssuedCommands int
		args               []string
	}{
		{"simple run", 0, 1, 100, []string{"-p", "6379", "-c", "10", "-n", "100", "HSET", "hash:1", "field", "value"}},
		{"run with rps", 0, 1, 100, []string{"-p", "6379", "-c", "10", "-n", "100", "-rps", "100", "HSET", "hash:1", "field", "value"}},
		{"run with key placeholder", 0, 10, 1000, []string{"-p", "6379", "-c", "10", "-n", "1000", "-r", "10", "-rps", "10000", "HSET", "hash:__key__", "field", "value"}},
		{"run with multiple commands", 0, 20, 1000, []string{"-p", "6379", "-c", "10", "-n", "1000", "-r", "10", "-rps", "10000", "-cmd", "HSET hash:__key__ field value", "-cmd-ratio", "0.5", "-cmd", "SET string:__key__ value", "-cmd-ratio", "0.5"}},
		{"bad run", 2, 0, 0, []string{"-p", "xx"}},
	}
	host, password := getTestConnectionDetails()
	clientOptions := rueidis.ClientOption{
		InitAddress:         []string{host},
		Password:            password,
		AlwaysPipelining:    false,
		AlwaysRESP2:         true,
		DisableCache:        true,
		BlockingPoolSize:    0,
		PipelineMultiplex:   0,
		RingScaleEachConn:   1,
		ReadBufferEachConn:  1024,
		WriteBufferEachConn: 1024,
	}
	clientOptions.Dialer.KeepAlive = time.Hour
	client, err := rueidis.NewClient(clientOptions)
	if err != nil {
		panic(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.Do(context.Background(), client.B().Flushall().Build()).Error()

			cmd := exec.Command("./redis-benchmark-go", tt.args...)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, "GOCOVERDIR=.coverdata")
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			exitCode := 0
			if err != nil {
				// try to get the exit code
				if exitError, ok := err.(*exec.ExitError); ok {
					ws := exitError.Sys().(syscall.WaitStatus)
					exitCode = ws.ExitStatus()
				}
			}
			if exitCode != tt.wantExitCode {
				t.Errorf("got exit code = %v, want %v", exitCode, tt.wantExitCode)
			}
			keyspaceLen := 0
			v, err := client.Do(context.Background(), client.B().Keys().Pattern("*").Build()).AsStrSlice()
			keyspaceLen = len(v)

			if keyspaceLen != tt.wantKeyspaceLength {
				t.Errorf("got keyspace length = %v, want %v", keyspaceLen, tt.wantKeyspaceLength)
			}
		})
	}

}
