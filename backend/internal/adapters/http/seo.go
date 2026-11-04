package http

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/refresco/refresco-v2/backend/internal/adapters/ogcard"
	d "github.com/refresco/refresco-v2/backend/internal/domain"
	"github.com/refresco/refresco-v2/backend/internal/ports"
)

// Apps that show a link preview, and most crawlers, read the HTML without running
// the site. nginx asks these routes for each page's details and writes them in.

const (
	siteName  = "Refresco"
	siteTitle = "Recebe apoios dos teus fãs por M-Pesa | Refresco"
	siteDesc  = "Um link para criadores, podcasts e artistas de Moçambique receberem apoios dos fãs por M-Pesa. O fã não precisa de app nem de registo, e levantas em até 72 horas."
)

type pageMeta struct {
	title, desc string
	image       string // a path under /og/
	alt         string
	path        string // where the page lives, for the canonical address
	index       bool
	ld          any
	noscript    string // what a reader without JavaScript gets
}

func (s *Server) url(path string) string {
	return strings.TrimRight(s.app.PublicURL, "/") + path
}

func (s *Server) metaFor(c context.Context, path string, q func(string) string) pageMeta {
	private := func(title string) pageMeta {
		return pageMeta{title: title + " | Refresco", desc: siteDesc, image: "/og/site.png", alt: siteTitle}
	}
	seg := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case path == "/" || path == "":
		return pageMeta{title: siteTitle, desc: siteDesc, image: "/og/site.png", alt: siteTitle, path: "/", index: true,
			ld: []any{
				map[string]any{"@context": "https://schema.org", "@type": "Organization", "name": siteName, "url": s.url("/"), "logo": s.url("/assets/refres-logo-orange.png"),
					"contactPoint": map[string]any{"@type": "ContactPoint", "telephone": "+258848255237", "contactType": "customer support", "areaServed": "MZ", "availableLanguage": "Portuguese"}},
				map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": siteName, "url": s.url("/"), "inLanguage": "pt-MZ"},
			},
			noscript: `<h1>Recebe apoios dos teus fãs por M-Pesa</h1><p>` + html.EscapeString(siteDesc) + `</p><p><a href="/registar">Criar a minha página</a></p>`}
	case seg[0] == "registar":
		if q("convite") != "" {
			return pageMeta{title: "Foste convidado para o Refresco", desc: "Cria a tua página e começa a receber apoios dos teus fãs por M-Pesa. O Refresco está aberto só a criadores convidados.", image: "/og/invite.png", alt: "Convite para o Refresco"}
		}
		return private("Criar conta")
	case seg[0] == "entrar":
		return private("Entrar")
	case seg[0] == "recuperar":
		return private("Recuperar palavra-passe")
	case seg[0] == "aprovar":
		return private("Aprovar divisão")
	case seg[0] == "painel" || seg[0] == "admin":
		return private("Conta")
	case len(seg) == 1:
		return s.creatorMeta(c, seg[0])
	}
	return private("Página não encontrada")
}

func (s *Server) creatorMeta(c context.Context, handle string) pageMeta {
	missing := pageMeta{title: "Página não encontrada | Refresco", desc: "Esta página não existe. Confirma o link com o criador.", image: "/og/site.png", alt: siteTitle}
	a, err := s.app.PublicPage(c, handle)
	if err != nil {
		return missing
	}
	title := "Apoia " + a.Name + " com um refresco | Refresco"
	desc := strings.TrimSpace(a.Bio)
	const how = "Apoia por M-Pesa: sem app nem registo, confirmas com o teu PIN."
	if desc == "" {
		desc = "Mostra o teu apoio a " + a.Name + ". " + how
	} else {
		desc = strings.TrimRight(desc, ". ") + ". " + how
	}
	var sameAs []string
	for _, u := range a.Socials {
		if strings.HasPrefix(u, "https://") {
			sameAs = append(sameAs, u)
		}
	}
	entity := map[string]any{"@type": "Organization", "name": a.Name, "url": s.url("/" + a.Handle)}
	if a.Bio != "" {
		entity["description"] = a.Bio
	}
	if len(sameAs) > 0 {
		entity["sameAs"] = sameAs
	}
	body := `<h1>Apoia ` + html.EscapeString(a.Name) + ` com um refresco</h1><p>` + html.EscapeString(desc) + `</p>`
	return pageMeta{title: title, desc: desc, image: "/og/p/" + a.Handle + ".png?v=" + version(a.Name, a.Bio), alt: "Apoia " + a.Name + " com um refresco",
		path: "/" + a.Handle, index: a.Status == d.AccountActive,
		ld:       map[string]any{"@context": "https://schema.org", "@type": "ProfilePage", "inLanguage": "pt-MZ", "url": s.url("/" + a.Handle), "mainEntity": entity},
		noscript: body}
}

// version changes the image's address when its content does, because WhatsApp
// keeps an image by its address.
func version(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:4])
}

func (m pageMeta) head(s *Server) string {
	e := html.EscapeString
	var b strings.Builder
	tag := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	tag(`<title>%s</title>`, e(m.title))
	tag(`<meta name="description" content="%s">`, e(trimDesc(m.desc)))
	robots := "noindex,nofollow"
	if m.index {
		robots = "index,follow"
		tag(`<link rel="canonical" href="%s">`, e(s.url(m.path)))
	}
	tag(`<meta name="robots" content="%s">`, robots)
	tag(`<meta property="og:site_name" content="%s">`, siteName)
	tag(`<meta property="og:locale" content="pt_MZ">`)
	tag(`<meta property="og:type" content="website">`)
	tag(`<meta property="og:title" content="%s">`, e(strings.TrimSuffix(m.title, " | Refresco")))
	tag(`<meta property="og:description" content="%s">`, e(trimDesc(m.desc)))
	if m.path != "" {
		tag(`<meta property="og:url" content="%s">`, e(s.url(m.path)))
	}
	tag(`<meta property="og:image" content="%s">`, e(s.url(m.image)))
	tag(`<meta property="og:image:type" content="image/png">`)
	tag(`<meta property="og:image:width" content="%d">`, ogcard.Width)
	tag(`<meta property="og:image:height" content="%d">`, ogcard.Height)
	tag(`<meta property="og:image:alt" content="%s">`, e(m.alt))
	tag(`<meta name="twitter:card" content="summary_large_image">`)
	if m.ld != nil {
		if j, err := json.Marshal(m.ld); err == nil {
			// In a script tag, "</script>" in a name would end it early.
			tag(`<script type="application/ld+json">%s</script>`, strings.ReplaceAll(string(j), "<", `<`))
		}
	}
	return b.String()
}

func trimDesc(s string) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= 200 {
		return string(r)
	}
	return strings.TrimSpace(string(r[:199])) + "…"
}

func (s *Server) metaHead(c *gin.Context) {
	m := s.metaFor(c.Request.Context(), c.Param("path"), c.Query)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(m.head(s)))
}

func (s *Server) metaBody(c *gin.Context) {
	m := s.metaFor(c.Request.Context(), c.Param("path"), c.Query)
	out := ""
	if m.noscript != "" {
		out = "<noscript>" + m.noscript + "</noscript>"
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(out))
}

func (s *Server) robots(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=3600")
	c.String(http.StatusOK, "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /painel\nDisallow: /api/\n\nSitemap: %s\n", s.url("/sitemap.xml"))
}

// The sitemap lists active creators, whose pages are what people search for.
func (s *Server) sitemap(c *gin.Context) {
	accounts, err := s.app.Store.Accounts().List(c.Request.Context(), ports.AccountFilter{Status: d.AccountActive})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	fmt.Fprintf(&b, "<url><loc>%s</loc></url>\n", html.EscapeString(s.url("/")))
	for _, a := range accounts {
		fmt.Fprintf(&b, "<url><loc>%s</loc></url>\n", html.EscapeString(s.url("/"+a.Handle)))
	}
	b.WriteString("</urlset>\n")
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(b.String()))
}

// Drawing takes milliseconds, but a shared link can be fetched in bursts.
type cardCache struct {
	mu    sync.Mutex
	items map[string][]byte
}

func (k *cardCache) get(key string, card ogcard.Card) ([]byte, error) {
	k.mu.Lock()
	b, ok := k.items[key]
	k.mu.Unlock()
	if ok {
		return b, nil
	}
	b, err := ogcard.Render(card)
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.items == nil || len(k.items) >= 200 {
		k.items = map[string][]byte{}
	}
	k.items[key] = b
	return b, nil
}

func (s *Server) ogImage(c *gin.Context) {
	path := strings.TrimSuffix(strings.Trim(c.Param("rest"), "/"), ".png")
	var card ogcard.Card
	switch {
	case path == "site":
		card = ogcard.Card{Title: "Os teus fãs pagam-te um refresco", Body: "Um link para receberes apoios por M-Pesa. Sem app e sem registo para o fã.", Big: "Para criadores de Moçambique", Foot: "pagarefresco.com"}
	case path == "invite":
		card = ogcard.Card{Kicker: "Foste convidado", Title: "Cria a tua página no Refresco", Body: "Recebe apoios dos teus fãs por M-Pesa.", Foot: "pagarefresco.com"}
	case strings.HasPrefix(path, "p/"):
		handle := strings.TrimPrefix(path, "p/")
		a, err := s.app.PublicPage(c.Request.Context(), handle)
		if errors.Is(err, d.ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		card = ogcard.Card{Kicker: "Paga um refresco a", Title: a.Name, Body: a.Bio, Big: "Apoia por M-Pesa", Foot: "pagarefresco.com/" + a.Handle}
	default:
		c.Status(http.StatusNotFound)
		return
	}
	b, err := s.cards.get(path+"|"+card.Title+"|"+card.Body, card)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Data(http.StatusOK, "image/png", b)
}
