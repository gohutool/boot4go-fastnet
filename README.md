# boot4go-fastnet
*a framework to start a web application quickly like as spring-boot*

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)


## INTRO

The boot4go Fastnet project is simple. The main code size is less than 20K, 
but its performance is not inferior to that of other network libraries. 
The idea of boot4go Fastnet to optimize network communication is mainly inspired by fasthttp, 
another high-performance HTTP service. 

Fasthttp only provides the communication implementation of HTTP, 
but it is not implemented at the TCP level. 


In fact, the co process method of go has very well solved the disadvantages of previous bio. 
The reason why NiO appears is that in the previous bio mode, there is a thread to process the 
connection obtained by listening. The thread maintenance cost is large. 

In the case of a large number of long connections, The overhead of the system will be very tight, 
so the NiO model is available. However, in go, the co processes are not as heavy as threads in dealing with a large 
number of concurrent cases. Looking at the IO model of redis, 
we all know that redis has excellent performance and multiplexing. 

There is such doubt, so I'd better test the performance by pressing the data first. 

I wrote a simple network communication library and implemented the following optimization points
according to the idea of fasthttp 

- introduce the process pool to further reduce the cost of the process


- memory reuse, cache pool and objects all use the implementation of object pool to further reduce the overhead
of memory allocation and GC


- zero copy. This is not really zero copy. It is only possible to reduce unnecessary memory 
copies through ByteBuffer.

## Performance testing

Take a look at the PK results of Fastnet and epoll.

Both of them are on my machine. 
When the concurrency number is 2000, the data is of great significance. 
When the concurrency number exceeds 2000, there will be some JMeter error reports and data 
distortion. The packets use less than 1K packets

- Results of snapshot:

### Fastnet

![image](https://img-blog.csdnimg.cn/1ec84bcc800a4daca2c3c30ebd6c71fd.png)


![image](https://img-blog.csdnimg.cn/29be341bb1674ea19fd4979cde246921.png)

### Epoll

![image](https://img-blog.csdnimg.cn/427de39a54b34a289fa26da74d04f1ae.png)


![image](https://img-blog.csdnimg.cn/67b1dc1250ec4ca182c215232f5283cf.png)

### golang/net

The implementation of the native golang net library is poor. In the case of 2000 concurrency,
JMeter is easy to report errors. In the case of 1000 concurrency, the data is meaningful

2000 data

![image](https://img-blog.csdnimg.cn/e50becca18b94b4cbeb9daf54d1f63e0.png)


![image](https://img-blog.csdnimg.cn/e5eda0cbbb8f49d3b91f25cd989802ab.png)


1000 data

![image](https://img-blog.csdnimg.cn/0a7f8547eb1842948b8c2100634b750b.png)


![image](https://img-blog.csdnimg.cn/7585918363754f5ab31831f5290a04ed.png)