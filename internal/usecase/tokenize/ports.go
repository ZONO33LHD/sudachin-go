// Package tokenize は形態素解析のユースケース (入力の正規化 → ラティス構築 → 経路探索 → 経路の書き換え → 分割) を実装する。
//
// 辞書やプラグインの具体的な実装には依存せず、このパッケージで定義したポート (インターフェース) だけを通して扱う。
// Tokenizer は複数 goroutine から同時に使われるので、ポートの実装も並行に呼ばれて安全でなければならない
// (呼び出しをまたいで状態を持たないこと)。
package tokenize

import (
	"iter"

	"github.com/ZONO33LHD/sudachin-go/internal/domain/lattice"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/text"
	"github.com/ZONO33LHD/sudachin-go/internal/domain/word"
)

// LexiconEntry は辞書の前方一致検索の 1 件。End は語の終端のバイト位置 (検索対象全体に対する絶対位置)。
type LexiconEntry struct {
	WordID word.ID
	End    int
}

// Lexicon は語彙辞書へのポート。
type Lexicon interface {
	// CommonPrefix は b[offset:] の接頭辞に一致する語を、短い順に列挙する。
	CommonPrefix(b []byte, offset int) iter.Seq[LexiconEntry]
	Param(id word.ID) word.Param
	Info(id word.ID) (word.Info, error)
}

// Grammar は品詞表と連接コスト表へのポート。
type Grammar interface {
	lattice.Coster
	POS(id uint16) (word.POS, bool)
}

// InputTextPlugin は解析前に入力テキストを書き換える。書き換えは必ず Editor 経由で行う。
type InputTextPlugin interface {
	Rewrite(current string, e *text.Editor) error
}

// OOVProvider は位置 offset (文字位置) から始まる未知語ノードを生成し、out に追加して返す。
// existing には辞書や先行するプロバイダがその位置に作った語の長さが入っている。
type OOVProvider interface {
	ProvideOOV(t *text.Text, offset int, existing CreatedWords, out []lattice.Node) []lattice.Node
}

// PathRewriter は最小コスト経路を書き換える (数詞の連結など)。引数の path は変更せず新しいスライスを返す。
type PathRewriter interface {
	Rewrite(t *text.Text, path []PathNode) ([]PathNode, error)
}
