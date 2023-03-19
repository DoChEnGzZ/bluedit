## 技术栈

1. gin框架
2. zap日志库
3. Viper配置管理
4. swagger生成文档
5. JWT认证
6. 令牌桶限流
7. Go语言操作MySQL
8. Go语言操作Redis

## 项目目录结构

```bash
.
├── conf
├── controller
├── dao
│   ├── mysql
│   └── redis
├── docs
├── log
├── logger
├── logic
├── middlewares
├── models
├── pkg
│   ├── jwt
│   └── snowflake
├── routers
├── settings
├── static
│   ├── css
│   ├── img
│   └── js
└── templates
```

## 项目预览图

![](./pic/vue.jpg)

## 压力测试

设置全局中间件，令牌桶令牌512个，分别用`go-stress-test`测试400、1024、2048个ping请求的执行情况。

```go
v1.Use(middlewares.RateLimitMiddleware(10*time.Millisecond,256))
func RateLimitMiddleware(fillInterval time.Duration, cap int64) func(c *gin.Context) {
	bucket := ratelimit.NewBucket(fillInterval, cap)
	return func(c *gin.Context) {
		// 如果取不到令牌就中断本次请求返回 rate limit...
		if bucket.TakeAvailable(1) == 0 {
			c.String(http.StatusOK, "rate limit...")
			c.Abort()
			return
		}
		// 取到令牌就放行
		c.Next()
	}
}
```

测试结果如下：可以看到

```bash
dochengzz@chengnizhideair go-stress-testing % go run main.go -c 2 -n 200 -u http://localhost:8081/ping

 开始启动  并发数:2 请求数:200 请求参数: 
request:
 form:http 
 url:http://localhost:8081/ping 
 method:GET 
 headers:map[Content-Type:application/x-www-form-urlencoded; charset=utf-8] 
 data: 
 verify:statusCode 
 timeout:30s 
 debug:false 
 http2.0：false 
 keepalive：false 
 maxCon:1 


─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────
 耗时│ 并发数│ 成功数│ 失败数│   qps  │最长耗时│最短耗时│平均耗时│下载字节│字节每秒│ 状态码
─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────
   0s│      2│    400│      0│ 1152.47│  109.21│    0.33│    1.74│   5,200│  14,632│200:400


*************************  结果 stat  ****************************
处理协程数量: 2
请求总数（并发数*请求数 -c * -n）: 400 总请求时间: 0.355 秒 successNum: 400 failureNum: 0
tp90: 1.000
tp95: 2.000
tp99: 23.000
*************************  结果 end   ****************************


dochengzz@chengnizhideair go-stress-testing % go run main.go -c 2 -n 512 -u http://localhost:8081/ping

 开始启动  并发数:2 请求数:512 请求参数: 
request:
 form:http 
 url:http://localhost:8081/ping 
 method:GET 
 headers:map[Content-Type:application/x-www-form-urlencoded; charset=utf-8] 
 data: 
 verify:statusCode 
 timeout:30s 
 debug:false 
 http2.0：false 
 keepalive：false 
 maxCon:1 


─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────
 耗时│ 并发数│ 成功数│ 失败数│   qps  │最长耗时│最短耗时│平均耗时│下载字节│字节每秒│ 状态码
─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────
   0s│      2│   1024│      0│ 3874.38│   14.89│    0.13│    0.52│  13,312│  48,190│200:1024


*************************  结果 stat  ****************************
处理协程数量: 2
请求总数（并发数*请求数 -c * -n）: 1024 总请求时间: 0.276 秒 successNum: 1024 failureNum: 0
tp90: 0.000
tp95: 0.000
tp99: 2.000
*************************  结果 end   ****************************


dochengzz@chengnizhideair go-stress-testing % go run main.go -c 2 -n 1024 -u http://localhost:8081/ping

 开始启动  并发数:2 请求数:1024 请求参数: 
request:
 form:http 
 url:http://localhost:8081/ping 
 method:GET 
 headers:map[Content-Type:application/x-www-form-urlencoded; charset=utf-8] 
 data: 
 verify:statusCode 
 timeout:30s 
 debug:false 
 http2.0：false 
 keepalive：false 
 maxCon:1 


─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────
 耗时│ 并发数│ 成功数│ 失败数│   qps  │最长耗时│最短耗时│平均耗时│下载字节│字节每秒│ 状态码
─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────
   1s│      2│   2048│      0│ 2479.81│  209.75│    0.15│    0.81│  26,624│  30,981│200:2048


*************************  结果 stat  ****************************
处理协程数量: 2
请求总数（并发数*请求数 -c * -n）: 2048 总请求时间: 0.859 秒 successNum: 2048 failureNum: 0
tp90: 0.000
tp95: 1.000
tp99: 2.000
*************************  结果 end   ****************************


dochengzz@chengnizhideair go-stress-testing % 

```

