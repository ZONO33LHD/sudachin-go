# sudachin-go

Sudachi 辞書 (SudachiDict) を使う、Go 製の日本語形態素解析器です。
[ikawaha/sudachi.go](https://github.com/ikawaha/sudachi.go) を参考に、クリーンアーキテクチャで一から組み直しました。

- 出力は sudachi.rs (Rust 版) と一致します。『坊っちゃん』『赤い蝋燭と人魚』計 2,671 文の A/B/C 全モードで一致することをゴールデンテストで確認しています。
- `Analyzer` は不変なので、1 つのインスタンスを複数 goroutine から同時に使えます。
- 外部依存は `golang.org/x/text` (NFKC 正規化) だけです。
- Go 1.27.1 以上が必要です (メソッドの型パラメータ、`encoding/json/v2` などを使っています)。`GOTOOLCHAIN=auto` (既定) なら古い Go からでも自動で 1.27.1 が使われます。

## 使い方

辞書は [SudachiDict](https://github.com/WorksApplications/SudachiDict) から入手します。

```sh
curl -LO https://d2ej7fkh96fzlu.cloudfront.net/sudachidict/sudachi-dictionary-latest-core.zip
unzip -j sudachi-dictionary-latest-core.zip '*.dic'

go run ./cmd/sudachin -l system_core.dic -a <<< "東京都に住んでいます"
```

| オプション | 意味 |
|---|---|
| `-l` | システム辞書のパス (省略時は `$SUDACHIN_DICT`) |
| `-m` | 分割単位 `A` / `B` / `C` (既定 `C`) |
| `-a` / `-w` / `-f json` | 全項目 / 分かち書き / JSON で出力 |
| `-o` | 出力ファイル |

ライブラリとして使う場合:

```go
a, err := sudachin.Open("system_core.dic")
if err != nil {
	return err
}
defer a.Close()

ms, err := a.Analyze("東京都に住んでいます", sudachin.ModeC)
for _, m := range ms {
	fmt.Println(m.Surface, m.POS, m.ReadingForm)
}
```

## 構成

```
sudachin.go                      コンポジションルート (依存関係の組み立て・公開 API)
cmd/sudachin                     CLI
internal/
  domain/                        外部に依存しないモデルとアルゴリズム
    word/  chars/  morpheme/     値オブジェクト (語 ID・品詞・文字種・形態素)
    text/                        正規化中の入力 (Builder) と解析用テキスト (Text)
    lattice/                     ラティスと Viterbi
  usecase/tokenize/              解析の手順とポート (Lexicon, Grammar, 各プラグイン)
  adapter/
    plugin/{inputtext,oov,pathrewrite}   ポートの実装 (正規化・未知語・数詞/カタカナ連結)
    presenter/                   出力形式
  infrastructure/
    dicfile/                     system.dic の読み込み (mmap)
    resource/                    char.def / unk.def / rewrite.def (go:embed)
```

依存は常に外側から内側 (infrastructure → adapter → usecase → domain) に向かいます。

## テスト

```sh
make check    # gofmt, go vet (darwin/windows), golangci-lint, staticcheck, govulncheck, go test -race

# sudachi.rs との互換性テスト (SudachiDict core 20250515 が必要)
make golden SUDACHIN_TEST_DICT=/path/to/20250515/system_core.dic

# ファジング
go test ./internal/usecase/tokenize -run '^$' -fuzz FuzzTokenize
```

lint ツールの版は `tools/go.mod` で固定しています (本体の `go.mod` には入りません)。
`gofmt` は、PATH 上の古いものではメソッドの型パラメータを解釈できないため、Makefile ではツールチェーン同梱の `$(go env GOROOT)/bin/gofmt` を使います。

## 未対応

- ユーザー辞書
- `sudachi.json` による設定の読み込み (既定の設定をコードで組み立てています)
- 文区切り (1 行を 1 文として解析します)

## ライセンス

`internal/infrastructure/resource/data` と `testdata` には Sudachi / sudachi.go 由来のファイルを含みます。詳細は [NOTICE](NOTICE) を参照してください。
