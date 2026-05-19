package i18n

// chitchatInputs holds the per-language exact-match input keywords used by
// the local intent matcher to detect bare social phrases (greeting, farewell,
// thanks). Reply variants for the same Phrase key live in the standard
// phrases map and are accessed via PickIn.
//
// Adding a new chitchat intent: add a Phrase const + entry here for input
// matchers + entry in phrases.go for reply variants + entry in
// ChitchatPhrases() so the intent matcher iterates it.
var chitchatInputs = map[Phrase]map[string][]string{
	PhraseChitchatGreeting: {
		LangVI:   {"chào", "chào lumi", "xin chào", "lumi ơi", "hey lumi"},
		LangEN:   {"hi", "hello", "hi lumi", "hello lumi", "hey", "hey lumi"},
		LangZhCN: {"你好", "你好啊", "嗨", "嘿"},
		LangZhTW: {"你好", "嗨"},
	},
	PhraseChitchatFarewell: {
		LangVI:   {"tạm biệt", "tạm biệt lumi"},
		LangEN:   {"bye", "bye lumi", "goodbye", "see you", "see ya", "later"},
		LangZhCN: {"再见", "拜拜"},
		LangZhTW: {"再見", "拜拜"},
	},
	PhraseChitchatThanks: {
		LangVI:   {"cảm ơn", "cảm ơn lumi"},
		LangEN:   {"thanks", "thank you", "thanks lumi", "thx"},
		LangZhCN: {"谢谢", "谢谢你"},
		LangZhTW: {"謝謝", "謝謝你"},
	},
}

// InputPhrases returns the per-language exact-match input keywords for the
// chitchat Phrase p. Returns nil when p isn't a chitchat phrase.
func InputPhrases(p Phrase) map[string][]string {
	return chitchatInputs[p]
}

// ChitchatPhrases returns the list of chitchat phrase keys in match order.
// intent.go iterates this so adding a new chitchat intent only needs i18n
// edits (Phrase const + phrases entry + chitchatInputs entry + this list).
func ChitchatPhrases() []Phrase {
	return []Phrase{
		PhraseChitchatGreeting,
		PhraseChitchatFarewell,
		PhraseChitchatThanks,
	}
}

// chitchatCommandWords are verbs/nouns per language that signal an action
// request, not a social phrase. The intent matcher rejects chitchat match
// when any of these appear in the input ("chào lumi bật đèn" → bật in VN
// command words → fall through to command rules so the LED toggle fires).
var chitchatCommandWords = map[string][]string{
	LangVI: {
		"bật", "tắt", "mở", "đóng", "phát", "dừng", "đổi", "chuyển",
		"chụp", "kể", "đọc", "hát", "hỏi", "tìm", "xem", "nói",
		"to lên", "nhỏ lại", "lớn hơn", "nhỏ hơn", "im lặng",
		"nhạc", "đèn", "ảnh",
	},
	LangEN: {
		"turn", "play", "stop", "switch", "change", "open", "close", "set",
		"show", "take", "tell", "read", "sing", "find", "search", "ask",
		"louder", "softer", "mute", "unmute", "lights", "music", "song",
	},
	LangZhCN: {"开", "关", "播放", "停", "换", "唱", "讲", "找", "拍", "看"},
	LangZhTW: {"開", "關", "播放", "停", "換", "唱", "講", "找", "拍", "看"},
}

// ChitchatCommandWords returns every command word across every supported
// language, flattened. Used by the intent matcher to reject chitchat on any
// command-bearing text regardless of which language the user is speaking.
func ChitchatCommandWords() []string {
	var out []string
	for _, ws := range chitchatCommandWords {
		out = append(out, ws...)
	}
	return out
}
