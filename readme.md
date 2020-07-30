# 提供服务

- 生成短链

  给定一个较长的 URL, 将其转换成较短版本, 比如:

  `http://maps.google.com/maps?f=q&source=s_q&hl=en&geocode=&q=tokyo&sll=37.0625,-95.677068&sspn=68.684234,65.566406&ie=UTF8&hq=&hnear=Tokyo,+Japan&t=h&z=9`

  将其转变为 `http://xxx/jUfhd` 并保存这一映射关系

- 重定向

  当访问短链接时, 会将用户重定向到原始的长 URL





# 对外提供 HTTP 接口

- GET `/` 生成短链页面
- GET `/del` 删除短链页面
- `/xxxx` 访问短链重定向到原始链接

> web 前端较为简陋



# 数据存储
分为两层
- 使用 map 来缓存数据
- 使用 redis 来持久化数据



短链服务是典型的极端读多写少, 因此可以通过水平扩展来增强性能, 大致架构如下:

```
http  request
|  |  |  |  |
V  V  V  V  V
 load balance
 |    |    |
 V    V    V
 s1   s2   s3
    |  |  |
    V  V  V
     redis
```

- 每个节点的地位是平等的
- 在从 redis 获取短链信息后各个节点会保留缓存
- "删除短链" 使用了 redis 的 subscribe channel, 每当一个节点删除短链时, 会通过 redis channel 发布通知, 每个节点订阅频道, 在收到删除消息时会同步删除自身缓存.





# 未解决

- [ ] 节点缓存短链信息过多导致内存溢出

  

