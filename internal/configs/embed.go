package configs

import _ "embed"

// IgnoreContent 是随二进制内嵌的默认排除规则，
// 内容与 install.sh 部署到 /usr/local/etc/wikitnow/ignore 的文件完全一致。
//
//go:embed default_ignore
var IgnoreContent []byte
