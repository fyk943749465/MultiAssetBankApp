package migrations

import _ "embed"

// SQL006Lending 为借贷模块 DDL + 种子数据（Base Sepolia）；启动时由 database 包执行。
//
//go:embed 006_lending.sql
var SQL006Lending string
