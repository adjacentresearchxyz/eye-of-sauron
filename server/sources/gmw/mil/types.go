package main

type GmwMilSource struct {
	Link    string
	Title   string
	Content string
}

type GmwMilSourceTranslated struct {
	Link            string
	OriginalTitle   string
	OriginalContent string
	EnglishTitle    string
	EnglishContent  string
}
