package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"time"

	"html"
	"strings"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"github.com/adrg/strutil/metrics"
)

type Source struct {
	ID                    int
	Title                 string
	Link                  string
	Date                  time.Time
	Summary               string
	ImportanceBool        bool
	ImportanceReasoning   string
	CreatedAt             time.Time
	Processed             bool
	RelevantPerHumanCheck string
}

var RELEVANT_PER_HUMAN_CHECK_NO = "no"
var RELEVANT_PER_HUMAN_CHECK_YES = "yes"
var RELEVANT_PER_HUMAN_CHECK_DEFAULT = "maybe"

type App struct {
	screen         tcell.Screen
	sources        []Source
	selectedIdx    int
	expandedItems  map[int]bool
	showImportance map[int]bool
	currentPage    int
	itemsPerPage   int
	failureMark    bool
	waitgroup      sync.WaitGroup
	statusMessage  string
}

type Topic struct {
	name     string
	keywords []string
}

func newApp() (*App, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %v", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %v", err)
	}

	return &App{
		screen:         screen,
		selectedIdx:    0,
		expandedItems:  make(map[int]bool),
		showImportance: make(map[int]bool),
		currentPage:    0,
		itemsPerPage:   10, // 17,
	}, nil
}

// Filtering
// Could eventually move to a new file
func readRegexesFromFile(filepath string) ([]*regexp.Regexp, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var regexes []*regexp.Regexp
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		regexStr := scanner.Text()
		if len(regexStr) > 1 && regexStr[0] != '#' {
			regex, err := regexp.Compile("(?i)" + regexStr) // make case insensitive
			if err != nil {
				return nil, err // exit at first failure
			}
			regexes = append(regexes, regex)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return regexes, nil
}

func testStringAgainstRegexes(rs []*regexp.Regexp, s string) bool {
	for _, r := range rs {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}

func isSourceRepeat(i int, sources []Source) bool {
	// TODO: maybe make this a bit more sophisticated, and mark as repeat even if they are not exactly the same but
	// also if they are pretty near according to some measure of distance
	for j := range sources {
		if i != j && strings.ToUpper(sources[i].Title) == strings.ToUpper(sources[j].Title) {
			return true
		}
	}
	return false
}

func filterSources(sources []Source) ([]Source, error) {
	var filtered_sources []Source
	regexes, err := readRegexesFromFile("src/filters.txt")
	if err != nil {
		log.Printf("Error loading regexes: %v", err)
		return filtered_sources, err
	}

	for i, source := range sources {
		match := testStringAgainstRegexes(regexes, source.Title)
		is_repeat := isSourceRepeat(i, sources) // TODO: maybe extract this into own loop
		if !match && !is_repeat {
			filtered_sources = append(filtered_sources, source)
		} else {
			log.Printf("Skipped over: %s", source.Title)
			go markProcessedInServer(true, source.ID, source)
		}
	}
	return filtered_sources, nil
}

func filterSourcesForUnread(sources []Source) []Source {
	var unread_sources []Source
	for _, source := range sources {
		if !source.Processed {
			unread_sources = append(unread_sources, source)
		} else {
		}
	}
	return unread_sources
}

func readTopicsFromFile(filepath string) ([]Topic, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var topics []Topic
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		keywords := strings.Split(parts[1], ",")
		// Clean up each keyword
		var cleanKeywords []string
		for _, k := range keywords {
			k = strings.TrimSpace(k)
			if k != "" {
				cleanKeywords = append(cleanKeywords, k)
			}
		}
		if len(cleanKeywords) > 0 {
			topics = append(topics, Topic{name: name, keywords: cleanKeywords})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return topics, nil
}

func reorderSources(sources []Source) ([]Source, error) {
	var reordered_sources []Source
	remaining_sources := sources

	topics, err := readTopicsFromFile("src/topics.txt")
	if err != nil {
		log.Printf("Error loading topics: %v", err)
		return sources, err
	}

	for _, topic := range topics {
		var topic_regexes []*regexp.Regexp
		for _, regex_string := range topic.keywords {
			regex, err := regexp.Compile("(?i)" + regex_string)
			if err != nil {
				log.Printf("Regex error: %v", err)
				return nil, err
			}
			topic_regexes = append(topic_regexes, regex)
		}

		var new_remaining_sources []Source
		var topic_sources []Source
		for _, source := range remaining_sources {
			match := testStringAgainstRegexes(topic_regexes, source.Title)
			if match {
				topic_sources = append(topic_sources, source)
			} else {
				new_remaining_sources = append(new_remaining_sources, source)
			}
		}

		// Sort the topic_sources alphabetically by title
		sort.Slice(topic_sources, func(i, j int) bool {
			return topic_sources[i].Title < topic_sources[j].Title
		})
		reordered_sources = append(reordered_sources, topic_sources...)
		remaining_sources = new_remaining_sources
	}

	// Append remaining sources that didn't fit into any topic, sorted alphabetically
	sort.Slice(remaining_sources, func(i, j int) bool {
		return remaining_sources[i].Title < remaining_sources[j].Title
	})
	reordered_sources = append(reordered_sources, remaining_sources...)

	return reordered_sources, nil
}

func skipSourcesWithSimilarityMetric(sources []Source) ([]Source, error) {
	if len(sources) < 2 {
		return sources, nil
	}

	new_sources := []Source{sources[0]}

	last_title := sources[0].Title
	for i := 1; i < len(sources); i++ {
		title_i := sources[i].Title
		if len(title_i) > 30 && len(last_title) > 30 {
			hamming := metrics.NewHamming()
			distance := hamming.Distance(title_i[:30], last_title[:30])
			if distance <= 4 {
				go markProcessedInServer(true, sources[i].ID, sources[i])
				continue
			}
			last_title = title_i
		}
		new_sources = append(new_sources, sources[i])
	}
	return new_sources, nil
}

func (a *App) loadSources() error {
	/* This syntax is a method in go <https://go.dev/tour/methods/8>
	the point is to pass a pointer
	so that you can avoid passing values around
	yet still be able to modify them
	while having somewhat terser syntax than a funcion that takes
	a pointer.
	At the same time, you could achieve a similar thing with a normal
	function.

	On top of that, you can define an interface, as a type that implements
	some method. <https://go.dev/tour/methods/10>
	*/
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_POOL_URL"))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, "SELECT id, title, link, date, summary, importance_bool, importance_reasoning, created_at, processed FROM sources WHERE processed = false ORDER BY date ASC, id ASC") // AND DATE_PART('doy', date) < 34
	// date '+%j'
	if err != nil {
		return fmt.Errorf("failed to query sources: %v", err)
	}
	defer rows.Close()

	var sources []Source
	for rows.Next() {
		var s Source
		err := rows.Scan(&s.ID, &s.Title, &s.Link, &s.Date, &s.Summary, &s.ImportanceBool, &s.ImportanceReasoning, &s.CreatedAt, &s.Processed)
		if err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}
		// Clean HTML entities and tags
		s.Title = stripHTML(html.UnescapeString(s.Title))
		s.Summary = stripHTML(html.UnescapeString(s.Summary))
		sources = append(sources, s)
	}

	filtered_sources, err := filterSources(sources)
	if err != nil {
		return nil
	}
	reordered_sources, err := reorderSources(filtered_sources)
	if err != nil {
		return nil
	}
	unsimilar_sources, err := skipSourcesWithSimilarityMetric(reordered_sources)
	if err != nil {
		return nil
	}
	a.sources = unsimilar_sources

	return nil
}

func padStringWithWhitespace(s string, n int) string {
	if len(s) > n {
		return s
	}
	padding := strings.Repeat(" ", n-len(s))
	return s + padding
}

func (a *App) draw() {
	a.screen.Clear()
	width, height := a.screen.Size()
	style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	// selectedStyle := tcell.StyleDefault.Background(tcell.ColorDarkBlue).Foreground(tcell.ColorWhite)
	selectedStyle := tcell.StyleDefault.Background(tcell.Color24).Foreground(tcell.ColorWhite)
	summaryStyle := style.Foreground(tcell.Color248)
	importanceStyle := style.Foreground(tcell.ColorYellow)

	startIdx := a.currentPage * a.itemsPerPage
	endIdx := startIdx + a.itemsPerPage
	if endIdx > len(a.sources) {
		endIdx = len(a.sources)
	}

	lineIdx := 0
	for idx := startIdx; idx < endIdx; idx++ {
		source := a.sources[idx]
		// Calculate total height needed for this item
		itemHeight := 1 // Title line
		if a.expandedItems[idx] && source.Summary != "" {
			summaryLines := (len(source.Summary) + width - 3) / (width - 2)
			itemHeight += summaryLines + 1
		}
		if a.showImportance[idx] && source.ImportanceReasoning != "" {
			importanceLines := (len(source.ImportanceReasoning) + width - 3) / (width - 2)
			itemHeight += importanceLines + 1
		}

		// Check if there's enough room for the entire item
		if lineIdx+itemHeight >= height-1 {
			break
		}

		currentStyle := style
		if idx == a.selectedIdx {
			currentStyle = selectedStyle
		}

		// Display title, domain & date
		processedMark := " "
		if source.Processed {
			processedMark = "x"
		}
		host := ""
		parsedURL, err := url.Parse(source.Link)
		if err != nil {
			host = ""
		} else {
			host = parsedURL.Host
		}

		// title := fmt.Sprintf("[%s] %s | %s | %s", processedMark, padStringWithWhitespace(source.Title, 85), padStringWithWhitespace(host, 30), source.Date.Format("2006-01-02")) // why isn't the padding here working???
		title := fmt.Sprintf("[%s] %s | %s | %s", processedMark, source.Title, host, source.Date.Format("2006-01-02")) // why isn't the padding here working???
		// title := "[" + processedMark + "] " + padStringWithWhitespace(source.Title, 85) + " | " + padStringWithWhitespace(host, 30) + " | " + source.Date.Format("2006-01-02")
		lineIdx = drawText(a.screen, 0, lineIdx, width, currentStyle, title)

		// If this is the selected item and we're in expanded mode, show the summary
		if a.expandedItems[idx] && source.Summary != "" {
			lineIdx++
			if lineIdx < height {
				lineIdx = drawText(a.screen, 2, lineIdx, width-2, summaryStyle, source.Summary)
			}
		}

		// Add importance reasoning display
		if a.showImportance[idx] && source.ImportanceReasoning != "" {
			lineIdx++
			if lineIdx < height {
				lineIdx = drawText(a.screen, 2, lineIdx, width-2, importanceStyle, "Importance: "+source.ImportanceReasoning)
			}
		}
		lineIdx++
	}

	// Draw help text at the bottom
	current_item := a.selectedIdx
	num_items := len(a.sources)
	num_pages := int(math.Ceil(float64(num_items) / float64(a.itemsPerPage)))
	helpText := fmt.Sprintf("^/v: Navigate (%d/%d) | <>: Change Page (%d/%d) | Enter: Expand/Collapse | I: Show Importance", current_item+1, num_items, a.currentPage+1, num_pages)
	helpText2 := "O: Open in Browser \n | M: Toggle mark | S: Save | Q: Quit"
	if a.statusMessage != "" {
		helpText2 = fmt.Sprintf("%s | %s", helpText2, a.statusMessage)
	} else if a.failureMark {
		helpText2 = helpText2 + " [database F]"
	}
	if height > 0 {
		drawText(a.screen, 0, height-2, width, style, helpText)
		drawText(a.screen, 0, height-1, width, style, helpText2)
	}

	a.screen.Show()
}

func markRelevantPerHumanCheckInServer(state string, id int) error {
	flag := true
	if flag {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_POOL_URL"))
		if err != nil {
			log.Printf("failed to connect to database: %v", err)
			return fmt.Errorf("database connection error: %v", err)
		}
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, "UPDATE sources SET relevant_per_human_check = $1 WHERE id = $2", state, id)
		if err != nil {
			log.Printf("failed to mark source as relevant: %v", err)
			return fmt.Errorf("database update error: %v", err)
		}
	}
	return nil
}

func (a *App) markRelevantPerHumanCheck(state string, i int) error {
	if len(a.sources) == 0 {
		return nil
	}

	// Toggle processed state in UI immediately
	a.sources[i].RelevantPerHumanCheck = state

	// Update database asynchronously
	a.waitgroup.Add(1)
	go func() {
		defer a.waitgroup.Done()
		err := markRelevantPerHumanCheckInServer(state, a.sources[i].ID)
		if err != nil {
			fmt.Printf("%v", err)
			go func() {
				a.failureMark = true
				time.Sleep(2)
				a.failureMark = false
			}()
			a.sources[i].RelevantPerHumanCheck = state
		}
	}()

	return nil
}

func markProcessedInServer(state bool, id int, source Source) error {
	flag := true
	if flag {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_POOL_URL"))
		if err != nil {
			log.Printf("failed to connect to database: %v", err)
			return fmt.Errorf("database connection error: %v", err)
		}
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, "UPDATE sources SET processed = $1 WHERE id = $2", state, id)
		if err != nil {
			log.Printf("failed to mark source as processed: %v", err)
			log.Printf("source: %v", source)
			return fmt.Errorf("database update error: %v", err)
		}
	}
	return nil
}

func (a *App) markProcessed(i int, source Source) error {
	if len(a.sources) == 0 {
		return nil
	}

	// Toggle processed state in UI immediately
	newState := !a.sources[i].Processed
	a.sources[i].Processed = newState

	// Update database asynchronously
	a.waitgroup.Add(1)
	go func() {
		defer a.waitgroup.Done()
		err := markProcessedInServer(newState, a.sources[i].ID, source)
		if err != nil {
			log.Printf("%v", err)
			go func() {
				a.failureMark = true
				time.Sleep(2)
				a.failureMark = false
			}()
			a.sources[i].Processed = !newState
		}
	}()

	//
	if a.sources[i].RelevantPerHumanCheck != RELEVANT_PER_HUMAN_CHECK_YES {
		a.markRelevantPerHumanCheck(RELEVANT_PER_HUMAN_CHECK_NO, i)
	}

	return nil
}

func (a *App) saveToFile(source Source) error {

	basePath := os.Getenv("MINUTES_FOLDER")

	now := time.Now()
	year, week := now.ISOWeek()
	dirName := fmt.Sprintf("%d-%02d", year, week)

	targetDir := filepath.Join(basePath, dirName)

	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	targetFile := filepath.Join(targetDir, "own.md")

	f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	data := fmt.Sprintf("\n%s\n%s\n%s\n", source.Title, source.Summary, source.Link)
	if _, err := f.Write([]byte(data)); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	// clip_bash_cmd := fmt.Sprintf("{ echo \"%s\"; echo \"%s\"; echo \"%s\"; } | /usr/bin/xclip -sel clip", source.Title, source.Summary, source.Link)
	// cmd := exec.Command("bash", "-c", clip_bash_cmd)
	// cmd.Run()
	// cmd := exec.Command("/usr/bin/tmux", "new-window", "bash -c -i \"mins\"")
	// cmd.Run()
	// nvim_cmd := fmt.Sprintf("nvim +$ %s", targetFile)
	nvim_cmd := fmt.Sprintf("nvim +'$-2' %s", targetFile)
	cmd := exec.Command("/usr/bin/tmux", "new-window", nvim_cmd)
	cmd.Run()

	return nil
}

// This is intentionally empty - title cleaning is now handled server-side
func cleanTitle(s string) string {
	return s
}

func (a *App) webSearch(source Source) {
	clean_title := cleanTitle(source.Title)
	web_search_bash_cmd := fmt.Sprintf("bash -i -c \"websearch \\\"%s\\\"\"", clean_title)
	cmd := exec.Command("/usr/bin/tmux", "new-window", web_search_bash_cmd)
	// log.Printf(web_search_bash_cmd)
	// cmd := exec.Command("/usr/bin/tmux", "new-window", "bash -c -i 'websearch 1 && bash'") //web_search_bash_cmd)
	cmd.Run()
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func (a *App) getInput(prompt string) string {
	// Clear bottom of screen
	width, height := a.screen.Size()
	style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	for i := 0; i < width; i++ {
		a.screen.SetContent(i, height-1, ' ', nil, style)
	}

	// Show prompt
	for i, r := range prompt {
		a.screen.SetContent(i, height-1, r, nil, style)
	}
	a.screen.Show()

	// Get input
	var input strings.Builder
	cursorPos := len(prompt)
	for {
		ev := a.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnter:
				return input.String()
			case tcell.KeyEscape:
				return ""
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if input.Len() > 0 {
					str := input.String()
					input.Reset()
					input.WriteString(str[:len(str)-1])
					a.screen.SetContent(cursorPos-1, height-1, ' ', nil, style)
					cursorPos--
				}
			case tcell.KeyRune:
				input.WriteRune(ev.Rune())
				a.screen.SetContent(cursorPos, height-1, ev.Rune(), nil, style)
				cursorPos++
			}
			a.screen.Show()
		}
	}
}

func (a *App) run() error {
	err := a.loadSources()
	if err != nil {
		return err
	}

	for {
		a.draw()

		switch ev := a.screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				return nil
			case tcell.KeyRight:
				if (a.currentPage+1)*a.itemsPerPage < len(a.sources) {
					a.currentPage++
					a.selectedIdx = a.currentPage * a.itemsPerPage
				}
			case tcell.KeyLeft:
				if a.currentPage > 0 {
					a.currentPage--
					a.selectedIdx = a.currentPage * a.itemsPerPage
				}
			case tcell.KeyRune:
				switch ev.Rune() {
				case 'q', 'Q':
					a.waitgroup.Wait() // gracefully wait for all goroutines to finish
					return nil
				case 'o', 'O':
					if len(a.sources) > 0 {
						openBrowser(a.sources[a.selectedIdx].Link)
					}
				case 'n', 'N':
					if a.selectedIdx < len(a.sources)-1 {
						a.selectedIdx++
					}
				case 'm', 'M', 'x':
					if len(a.sources) > 0 {
						a.markProcessed(a.selectedIdx, a.sources[a.selectedIdx])
						if a.selectedIdx < len(a.sources)-1 && (a.selectedIdx+1) < (a.currentPage+1)*a.itemsPerPage {
							a.selectedIdx++
						} else if (a.currentPage+1)*a.itemsPerPage < len(a.sources) {
							a.currentPage++
							a.selectedIdx = a.currentPage * a.itemsPerPage
						}
					}
				case 'X', 'p':
					startIdx := a.currentPage * a.itemsPerPage
					endIdx := startIdx + a.itemsPerPage
					if endIdx > len(a.sources) {
						endIdx = len(a.sources)
					}
					for idx := startIdx; idx < endIdx; idx++ {
						a.markProcessed(idx, a.sources[a.selectedIdx])
					}
					if (a.currentPage+1)*a.itemsPerPage < len(a.sources) {
						a.currentPage++
						a.selectedIdx = a.currentPage * a.itemsPerPage
					}
				case 'r':
					a.screen.Clear()
					a.screen.Show()
					a.currentPage = 0
					a.selectedIdx = 0
					for i := range a.expandedItems {
						a.expandedItems[i] = false
						a.showImportance[i] = false
					}

					a.sources = filterSourcesForUnread(a.sources)
				case 'R':
					a.screen.Clear()
					a.screen.Show()
					a.currentPage = 0
					a.selectedIdx = 0
					a.loadSources()
				case 's', 'S':
					if len(a.sources) > 0 {
						a.saveToFile(a.sources[a.selectedIdx])
					}
					a.markRelevantPerHumanCheck(RELEVANT_PER_HUMAN_CHECK_YES, a.selectedIdx)
				case 'w', 'W':
					a.webSearch(a.sources[a.selectedIdx])
				case 'f', 'F':
					// Add new filter
					filter_input := a.getInput("Enter filter keyword: ")
					if filter_input != "" {
						a.statusMessage = "Filtering items..."
						a.draw()

						// Add to filters file
						// f, err := os.OpenFile("src/filters.txt", os.O_APPEND|os.O_WRONLY, 0644)
						// if err == nil {
						// 	_, err = f.WriteString("\n" + filter)
						// 	f.Close()
						// }
						// if err != nil {
						// 		log.Printf("Error writing filter: %v", err)
						// }

						filterRegex, err := regexp.Compile("(?i)" + filter_input)
						if err != nil {
							log.Printf("Error compiling regex: %v", err)
							continue
						}

						// Filter items locally and mark them in server
						var remaining_sources []Source
						for _, source := range a.sources {
							if filterRegex.MatchString(source.Title) {
								go markProcessedInServer(true, source.ID, source)
							} else {
								remaining_sources = append(remaining_sources, source)
							}
						}
						a.sources = remaining_sources

						// Reset page if needed
						if a.selectedIdx >= len(a.sources) {
							a.selectedIdx = len(a.sources) - 1
						}
						if a.selectedIdx < 0 {
							a.selectedIdx = 0
						}
						a.currentPage = a.selectedIdx / a.itemsPerPage

						// Clear status message
						a.statusMessage = ""
						a.draw()
					}
				case 'i', 'I':
					if len(a.sources) > 0 {
						a.showImportance[a.selectedIdx] = !a.showImportance[a.selectedIdx]
					}
				}
			case tcell.KeyUp:
				if a.selectedIdx > 0 {
					a.selectedIdx--
				}
			case tcell.KeyDown:
				if a.selectedIdx < len(a.sources)-1 && (a.selectedIdx+1) < (a.currentPage+1)*a.itemsPerPage {
					a.selectedIdx++
				}
			case tcell.KeyEnter:
				a.expandedItems[a.selectedIdx] = !a.expandedItems[a.selectedIdx]
				a.showImportance[a.selectedIdx] = false
			}
		case *tcell.EventResize:
			a.screen.Sync()
		}
	}
}

func drawText(screen tcell.Screen, x, y, maxWidth int, style tcell.Style, text string) int {
	words := strings.Fields(text)
	if len(words) == 0 {
		return y
	}

	currentLine := words[0]
	currentY := y

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= maxWidth {
			currentLine += " " + word
		} else {
			// Draw current line
			for i, r := range currentLine {
				screen.SetContent(x+i, currentY, r, nil, style)
			}
			currentY++
			currentLine = word
		}
	}

	// Draw final line
	for i, r := range currentLine {
		screen.SetContent(x+i, currentY, r, nil, style)
	}

	return currentY
}

func stripHTML(s string) string {
	var result strings.Builder
	var inTag bool
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func main() {
	logFile, err := os.OpenFile("src/client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()
	mw := io.Writer(logFile)
	log.SetOutput(mw)

	// Load environment variables from root directory
	err = godotenv.Load("../../.env")  // Look for .env in root directory
	if err != nil {
		err = godotenv.Load("../.env")  // Try one level up
		if err != nil {
			err = godotenv.Load(".env")  // Try current directory as fallback
			if err != nil {
				log.Fatal("Error loading .env file")
			}
		}
	}

	app, err := newApp()
	if err != nil {
		log.Fatalf("Could not create app: %v", err)
	}

	if err := app.run(); err != nil {
		app.screen.Fini()
		log.Fatalf("Error running app: %v", err)
	}

	app.screen.Fini()
}
