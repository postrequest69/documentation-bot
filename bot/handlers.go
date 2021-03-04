package bot

import (
	"fmt"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/postrequest69/dr-docso/docs"
)

// DocsHelpEmbed - the embed to give help to the docs command.
var DocsHelpEmbed = &discordgo.MessageEmbed{
	Title: "Docs help!",
}

// HandleDoc - Here is the replacement for the current system
func HandleDoc(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	var msg *discordgo.MessageEmbed
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // only the invocation
		msg = helpShortResponse()
	case 2: // invocation + arg
		msg = pkgResponse(fields[1])
	case 3: // invocation + pkg + func
		if strings.Contains(fields[2], ".") {
			split := strings.Split(fields[2], ".")
			msg = methodResponse(fields[1], split[0], split[1])
		} else {
			msg = funcResponse(fields[1], fields[2])
		}
	case 4:
		switch strings.ToLower(fields[2]) {
		case "func", "function", "fn":
			if strings.Contains(fields[3], ".") {
				split := strings.Split(fields[3], ".")
				msg = methodResponse(fields[1], split[0], split[1])
			} else {
				msg = funcResponse(fields[1], fields[3])
			}
		case "type":
			msg = typeResponse(fields[1], fields[3])
		default:
			msg = errResponse("Unsupported search type %q\nValid options are:\n\t`func`\n\t`type`", fields[2])
		}
	default:
		msg = errResponse("Too many arguments.")
	}

	if msg == nil {
		msg = errResponse("No results found, possibly an internal error.")
	}
	s.ChannelMessageSendEmbed(m.ChannelID, msg)
}

func funcResponse(pkg, name string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occurred while fetching the page for pkg `%s`", pkg)
	}
	if len(doc.Functions) == 0 {
		return errResponse("No results found for package: %q, function: %q", pkg, name)
	}

	var msg string

	// TODO(note): maybe use levenshtein here?
	for _, fn := range doc.Functions {
		if fn.Type == docs.FnNormal && strings.EqualFold(fn.Name, name) {
			// match found
			name = fn.Name
			msg += fmt.Sprintf("`%s`", fn.Signature)
			if len(fn.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", fn.Comments[0])
			} else {
				msg += "\n*no information*"
			}
			if fn.Example != "" {
				msg += fmt.Sprintf("\n\nExample:\n```go\n%s\n```", fn.Example)
			}
		}
	}

	if msg == "" {
		return errResponse("The package `%s` does not have function `%s`", pkg, name)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message was trimmed to fit the 2k character limit*", msg[:1950])
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: func %s", pkg, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%v#%v", doc.URL, name),
		},
	}
}

func errResponse(format string, args ...interface{}) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "Error",
		Description: fmt.Sprintf(format, args...),
	}
}

func typeResponse(pkg, name string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occurred while getting the page for the package `%s`", pkg)
	}
	if len(doc.Types) == 0 {
		return errResponse("Package `%s` seems to have no type definitions", pkg)
	}

	var msg string

	for _, t := range doc.Types {
		if strings.EqualFold(t.Name, name) {
			// got a match

			// To get the hyper link (case it's case sensitive)
			name = t.Name
			msg += fmt.Sprintf("```go\n%s\n```", t.Signature)
			if len(t.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", t.Comments[0])
			} else {
				msg += "\n*no information*"
			}
		}
	}

	if msg == "" {
		return errResponse("Package `%s` does not have type `%s`", pkg, name)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message is trimmed to fit the 2k character limit*", msg[:1950])
	}

	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: type %s", pkg, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%v#%v", doc.URL, name),
		},
	}
}

func helpShortResponse() *discordgo.MessageEmbed {
	return DocsHelpEmbed
}

func pkgResponse(pkg string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("An error occured when requesting the page for the package `%s`", pkg)
	}
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Info for %s", pkg),
		Description: fmt.Sprintf("Types: %v\nFunctions:%v", len(doc.Types), len(doc.Functions)),
	}
	return embed
}

func methodResponse(pkg, t, name string) *discordgo.MessageEmbed {
	doc, err := getDoc(pkg)
	if err != nil {
		return errResponse("Error while getting the page for the package `%s`", pkg)
	}
	if len(doc.Functions) == 0 {
		return errResponse("Package `%s` seems to have no functions", pkg)
	}

	var msg string

	for _, fn := range doc.Functions {
		if fn.Type == docs.FnMethod &&
			strings.EqualFold(fn.Name, name) &&
			strings.EqualFold(fn.MethodOf, t) {
			name = fn.Name
			msg += fmt.Sprintf("`%s`", fn.Signature)
			if len(fn.Comments) > 0 {
				msg += fmt.Sprintf("\n%s", fn.Comments[0])
			} else {
				msg += "\n*no info*"
			}
			if fn.Example != "" {
				msg += fmt.Sprintf("\nExample:\n```\n%s\n```", fn.Example)
			}
		}
	}
	if msg == "" {
		return errResponse("Package `%s` does not have `func(%s) %s`", pkg, t, name)
	}
	if len(msg) > 2000 {
		msg = fmt.Sprintf("%s\n\n*note: the message is trimmed to fit the 2k character limit*", msg[:1950])
	}
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s: func(%s) %s", pkg, t, name),
		Description: msg,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%v#%v", doc.URL, name),
		},
	}
}

// getDoc is a wrapper for docs.GetDoc that also implements caching for stdlib packages.
func getDoc(pkg string) (*docs.Doc, error) {
	var err error
	doc, ok := StdlibCache[pkg]
	if !ok || doc == nil {
		doc, err = docs.GetDoc(pkg)
		if err != nil {
			return nil, err
		}
	}

	// cache the stdlib pkg
	if ok && doc != nil {
		StdlibCache[pkg] = doc
	}
	return doc, nil
}

// PagesShortResponse - basically just a help command for the pages system :p
func PagesShortResponse(state, prefix string) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Help %v", state),
		Description: fmt.Sprintf("It seems you didn't have enough arguments, so here's an example!\n\n%v%v strings", prefix, state),
	}
}

// FuncsPages - for the reaction pages with all the functions in a package!
func FuncsPages(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // send a help command here
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("getfuncs", prefix))
		return
	case 2: // command + pkg (send page if possible)
		//TODO impl this
		doc, err := getDoc(fields[1])
		if err != nil || doc == nil {
			s.ChannelMessageSendEmbed(m.ChannelID, errResponse("Error while getting the page for the package `%s`", fields[1]))
			return
		}
		var pageLimit = int(math.Ceil(float64(len(doc.Functions)) / 10.0))
		var page = &ReactionListener{
			Type:        "functions",
			CurrentPage: 1,
			PageLimit:   pageLimit,
			UserID:      m.Author.ID,
			Data:        doc,
			LastUsed:    MakeTimestamp(),
		}
		textTosend := formatForMessage(page)
		m, err := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "functions",
			Description: textTosend,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Page 1/" + fmt.Sprint(pageLimit),
			},
		})
		if err != nil {
			return
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, leftArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, rightArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, destroyEmoji)
		pageListeners[m.ID] = page
		return
	default: // too many arguments
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("getfuncs", prefix))
		return
	}

}

// TypesPages - for the reaction pages with all the types in a package!
func TypesPages(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string, prefix string) {
	fields := strings.Fields(m.Content)
	switch len(fields) {
	case 0: // probably impossible
		return
	case 1: // send a help command here
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("gettypes", prefix))
		return
	case 2: // command + pkg (send page if possible)
		//TODO impl this
		doc, err := getDoc(fields[1])
		if err != nil || doc == nil {
			s.ChannelMessageSendEmbed(m.ChannelID, errResponse("Error while getting the page for the package `%s`", fields[1]))
			return
		}
		var pageLimit = int(math.Ceil(float64(len(doc.Types)) / 10.0))
		var page = &ReactionListener{
			Type:        "types",
			CurrentPage: 1,
			PageLimit:   pageLimit,
			UserID:      m.Author.ID,
			Data:        doc,
			LastUsed:    MakeTimestamp(),
		}
		textTosend := formatForMessage(page)
		m, err := s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Title:       "types",
			Description: textTosend,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Page 1/" + fmt.Sprint(pageLimit),
			},
		})
		if err != nil {
			return
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, leftArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, rightArrow)
		s.MessageReactionAdd(m.ChannelID, m.ID, destroyEmoji)
		pageListeners[m.ID] = page
		return
	default: // too many arguments
		s.ChannelMessageSendEmbed(m.ChannelID, PagesShortResponse("gettypes", prefix))
		return
	}
}
