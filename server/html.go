package server

type HTMLFile string

const (
	// HTML Base Templates
	RootHTML   HTMLFile = "templates/root.html"
	HeadHTML   HTMLFile = "templates/head.html"
	LayoutHTML HTMLFile = "templates/layout.html"

	// HTML Views
	TodayHTML    HTMLFile = "templates/views/today.html"
	HistoryHTML  HTMLFile = "templates/views/history.html"
	LoginHTML    HTMLFile = "templates/views/login.html"
	RegisterHTML HTMLFile = "templates/views/register.html"

	// HTML Components
	NavHTML            HTMLFile = "templates/components/nav.html"
	TableHTMLComponent HTMLFile = "templates/components/table.html"
	ModalHTMLComponent HTMLFile = "templates/components/modal.html"
)

// Views
var LoginView = []HTMLFile{
	RootHTML,
	LayoutHTML,
	HeadHTML,
	NavHTML,
	LoginHTML,
}

var RegisterView = []HTMLFile{
	RootHTML,
	LayoutHTML,
	HeadHTML,
	NavHTML,
	RegisterHTML,
}

var TodayView = []HTMLFile{
	HeadHTML,
	LayoutHTML,
	RootHTML,
	NavHTML,
	TodayHTML,
	TableHTMLComponent,
	ModalHTMLComponent,
}

var HistoryView = []HTMLFile{
	HeadHTML,
	LayoutHTML,
	RootHTML,
	NavHTML,
	HistoryHTML,
	TableHTMLComponent,
}
