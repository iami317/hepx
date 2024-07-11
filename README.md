# frp

[![Build Status](https://circleci.com/gh/fatedier/frp.svg?style=shield)](https://circleci.com/gh/fatedier/frp)
[![GitHub release](https://img.shields.io/github/tag/fatedier/frp.svg?label=release)](https://github.com/iami317/hepx/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/iami317/hepx)](https://goreportcard.com/report/github.com/iami317/hepx)
[![GitHub Releases Stats](https://img.shields.io/github/downloads/fatedier/frp/total.svg?logo=github)](https://somsubhra.github.io/github-release-stats/?username=fatedier&repository=frp)

[README](README.md) | [中文文档](README_zh.md)

frp 是一个专注于内网穿透的高性能的反向代理应用，支持 TCP、UDP、HTTP、HTTPS 等多种协议，且支持 P2P 通信。可以将内网服务以安全、便捷的方式通过具有公网 IP 节点的中转暴露到公网。

<h3 align="center">Gold Sponsors</h3>
<!--gold sponsors start-->
<p align="center">
  <a href="https://workos.com/?utm_campaign=github_repo&utm_medium=referral&utm_content=frp&utm_source=github" target="_blank">
    <img width="350px" src="https://raw.githubusercontent.com/fatedier/frp/dev/doc/pic/sponsor_workos.png">
  </a>
  <a>&nbsp</a>
  <a href="https://github.com/daytonaio/daytona" target="_blank">
    <img width="360px" src="https://raw.githubusercontent.com/fatedier/frp/dev/doc/pic/sponsor_daytona.png">
  </a>
</p>
<!--gold sponsors end-->

## 为什么使用 frp ？

通过在具有公网 IP 的节点上部署 frp 服务端，可以轻松地将内网服务穿透到公网，同时提供诸多专业的功能特性，这包括：

* 客户端服务端通信支持 TCP、QUIC、KCP 以及 Websocket 等多种协议。
* 采用 TCP 连接流式复用，在单个连接间承载更多请求，节省连接建立时间，降低请求延迟。
* 代理组间的负载均衡。
* 端口复用，多个服务通过同一个服务端端口暴露。
* 支持 P2P 通信，流量不经过服务器中转，充分利用带宽资源。
* 多个原生支持的客户端插件（静态文件查看，HTTPS/HTTP 协议转换，HTTP、SOCK5 代理等），便于独立使用 frp 客户端完成某些工作。
* 高度扩展性的服务端插件系统，易于结合自身需求进行功能扩展。
* 服务端和客户端 UI 页面。



## 文档

完整文档已经迁移至 [https://gofrp.org](https://gofrp.org)。

## 为 frp 做贡献

frp 是一个免费且开源的项目，我们欢迎任何人为其开发和进步贡献力量。

* 在使用过程中出现任何问题，可以通过 [issues](https://github.com/iami317/hepx/issues) 来反馈。
* Bug 的修复可以直接提交 Pull Request 到 dev 分支。
* 如果是增加新的功能特性，请先创建一个 issue 并做简单描述以及大致的实现方法，提议被采纳后，就可以创建一个实现新特性的 Pull Request。
* 欢迎对说明文档做出改善，帮助更多的人使用 frp，特别是英文文档。
* 贡献代码请提交 PR 至 dev 分支，master 分支仅用于发布稳定可用版本。
* 如果你有任何其他方面的问题或合作，欢迎发送邮件至 fatedier@gmail.com 。

**提醒：和项目相关的问题请在 [issues](https://github.com/iami317/hepx/issues) 中反馈，这样方便其他有类似问题的人可以快速查找解决方法，并且也避免了我们重复回答一些问题。**

## 关联项目

* [gofrp/plugin](https://github.com/gofrp/plugin) - frp 插件仓库，收录了基于 frp 扩展机制实现的各种插件，满足各种场景下的定制化需求。
* [gofrp/tiny-frpc](https://github.com/gofrp/tiny-frpc) - 基于 ssh 协议实现的 frp 客户端的精简版本(最低约 3.5MB 左右)，支持常用的部分功能，适用于资源有限的设备。
