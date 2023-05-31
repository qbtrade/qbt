# qbt
qb tool


# Installation

```
go install  github.com/qbtrade/qbt@v0.1.11
```

# Usage

## Monitor TCP connection latency and stability

```
--timeout TCP connect timeout
--count max count 
--interval interval between each connection
```

```
qbt monitor-tcp --timeout 2 --count 10000 --interval 1.5 10.110.1.86:22
```
