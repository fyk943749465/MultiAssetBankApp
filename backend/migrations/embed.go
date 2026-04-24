package migrations

import _ "embed"

// SQL006Lending 为借贷模块 DDL + 种子数据（Base Sepolia）；启动时由 database 包执行。
//
//go:embed 006_lending.sql
var SQL006Lending string

// SQL007Lending 为借贷模块增量：子图新增实体对应的 PG 表、contract_kind 扩展、默认合约地址刷新。
//
//go:embed 007_lending.sql
var SQL007Lending string
