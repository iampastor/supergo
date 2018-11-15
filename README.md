# supergo

supergo 是一个进程管理工具，同supervisor类似，但是主要的作用是为进程提供一种平滑重启的方式

supergo在启动进程时，将进程需要的文件描述符传递给子进程，子进程直接通过此listener进行连接的监听，在重启时，首先使用同一个listener启动另一个子进程，
然后将之前的进程KILL，保证重启的过程中，新的连接会由新的子进程来处理，而不会出现连接拒绝的情况

在平滑重启的时候，需要考虑到可能会同时存在两个进程的情况的影响

## 启动运行

`supergo`启动方式同普通的Go程序一样，启动之后在前台运行，可以通过其他的进程管理工具等方式，将其后台运行
```bash
supergo --help

Usage of bin/supergo:
  -config string
    	supervisord config file path (default "config/supergo.toml")
  -listen string
    	listen address (default "127.0.0.1:22106")
  -version
    	print version info & exit
```
`supergo`在启动时，会去自动加载配置文件中所有的进程配置，并将其启动，可以通过`supergoctl`对其进行管理:  
```bash
supergoctl

Usage of supergoctl:

-url 127.0.0.1:22106 // supergo的地址

Commands:

supergoctl status
supergoctl reread
supergoctl update
supergoctl start <prog>
supergoctl stop <prog>
supergoctl restart <prog>
```

## 程序使用示例

一个程序想要通过supergo的方式来管理，需要将原来自己监听端口的方式，改为通过文件listener的方式，同时，在退出时，关闭该listener：  
```go
var wg sync.WaitGroup
// 文件描述符从3开始
fileListener := os.NewFile(3, "ghttpserver")
if fileListener == nil {
    log.Panic("invalid fd")
} else {
    l, err = net.FileListener(fileListener)
    if err != nil {
        log.Panic(err)
    }
    // 关闭文件描述符
    fileListener.Close()
    httpServer := &http.Server{}
    http.HandleFunc("/hello", func(resp http.ResponseWriter, req *http.Request) {
        wg.Add(1)
        defer wg.Done()

        name := req.FormValue("name")
        if name == "" {
            name = "Jack"
        }
        resp.Write([]byte(fmt.Sprintf("hello, %s", name)))
    })

    // 使用新建的listener启动
    err = httpServer.Serve(l)
    if err != nil {
        log.Printf("metrics server: %s", err.Error())
    }

    // 进程在手动停止或者重启时，将会收到TERM信号量，可以在收到信号量之后，关闭listener，继续处理之前的连接，
    // 新的连接将会由新启动的进程来处理
    sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	sig := <-sigCh
	switch sig {
    case syscall.SIGTERM:
        // 关闭listener
        l.Close()
        // 等待所有的连接处理完成
		wg.Wait()
	}
}
```

## 进程配置

配置文件的格式为toml，可参考`config/supergo.toml`，详情如下：
```toml
[program.test]
directory = "/home/www" # 进程运行的目录
command = " ./http_listener -arg=test world" # 运行的指令
auto_restart = true # 是否自动重启
stdout_file = "/tmp/hello.log" # 进程的标准输出，为空将不会输出
stderr_file = "/tmp/hello.err" # 进程的标准错误输出，为空将不会输出
max_retry = 3 # 重启的次数
listen_addrs = [":4041"] # 进程需要监听的端口，从文件描述符3开始，可以为空
stop_timeout = 10 # 重启时，将发送TRERM信号，如果超时进程还没有退出，将强行KILL
stop_before_restart = false # 重启时是否先停止老的进程，默认为false，既会先启动一个新的进程，再停止老的进程
[include]
files = "config/conf.d/*.toml"
```

## TODO
- [ ] 配置文件的检查和错误提示
- [ ] 进程可配置的内容更多，例如进程运行的用户、退出时接受的信号量等