package email

type Input struct {
	Recipient string
	Facts     []string
	Goal      string
	Tone      Tone
	Language  Language
}

type Tone string

const (
	ToneConcise  Tone = "concise"
	ToneFormal   Tone = "formal"
	ToneFriendly Tone = "friendly"
)

func Tones() []Tone {
	return []Tone{ToneConcise, ToneFormal, ToneFriendly}
}

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageSpanish Language = "es"
)

func Languages() []Language {
	return []Language{LanguageEnglish, LanguageSpanish}
}

type Draft struct {
	Subject    string
	Body       string
	Tone       Tone
	Language   Language
	Claims     []Claim
	Unresolved []string
}

type Claim struct {
	Text string
	Fact int
}

type Problem struct {
	Field   string
	Message string
}
