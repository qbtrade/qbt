# qbt
qb tool


# Installation

```
go get  github.com/qbtrade/qbt@v0.1.1
```


# Usage

## Monitor TCP connection latency and stability

```
qbt monitor-tcp --timeout 2 --count 10000 --interval 2 10.110.1.86:22
```
