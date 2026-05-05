package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// SiteDataDocument is the full validated website JSON payload stored under siteData.data.
type SiteDataDocument struct {
	Meta     SiteMeta     `json:"meta"`
	Nav      SiteNav      `json:"nav"`
	Hero     SiteHero     `json:"hero"`
	Ventures SiteVentures `json:"ventures"`
	About    SiteAbout    `json:"about"`
	Skills   SiteSkills   `json:"skills"`
	Contact  SiteContact  `json:"contact"`
	Footer   SiteFooter   `json:"footer"`
}

type SiteMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	SiteURL     string `json:"siteUrl"`
	BrandName   string `json:"brandName"`
}

type SiteNav struct {
	Items []SiteNavItem `json:"items"`
}

type SiteNavItem struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type SiteHero struct {
	Pill              string        `json:"pill"`
	Name              string        `json:"name"`
	Taglines          []string      `json:"taglines"`
	TaglineIntervalMs int           `json:"taglineIntervalMs"`
	CTAs              []SiteHeroCTA `json:"ctas"`
}

type SiteHeroCTA struct {
	ID           string         `json:"id"`
	Label        string         `json:"label"`
	Variant      string         `json:"variant"`
	Size         string         `json:"size"`
	Action       SiteHeroAction `json:"action"`
	TrailingIcon string         `json:"trailingIcon"`
}

type SiteHeroAction struct {
	Type     string `json:"type"`
	TargetID string `json:"targetId"`
}

type SiteVentures struct {
	Heading SiteHeadingTriple `json:"heading"`
	UI      SiteVenturesUI    `json:"ui"`
	Seed    []SiteVentureSeed `json:"seed"`
}

type SiteHeadingTriple struct {
	Eyebrow     string `json:"eyebrow"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type SiteVenturesUI struct {
	StatusLabels      SiteVentureStatusLabels `json:"statusLabels"`
	ImagePlaceholder  string                  `json:"imagePlaceholder"`
	CarouselAriaLabel string                  `json:"carouselAriaLabel"`
	SkeletonCount     int                     `json:"skeletonCount"`
}

type SiteVentureStatusLabels struct {
	Live       string `json:"live"`
	ComingSoon string `json:"comingSoon"`
}

type SiteVentureSeed struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	LongDescription string  `json:"longDescription"`
	Status          string  `json:"status"`
	ImageURL        *string `json:"imageUrl"`
	Href            string  `json:"href"`
}

type SiteAbout struct {
	Heading SiteHeadingTriple `json:"heading"`
	Cards   []SiteAboutCard   `json:"cards"`
}

type SiteAboutCard struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type SiteSkills struct {
	Heading SiteHeadingTriple `json:"heading"`
	Items   []string          `json:"items"`
	Marquee SiteSkillsMarquee `json:"marquee"`
}

type SiteSkillsMarquee struct {
	DurationSeconds int `json:"durationSeconds"`
}

type SiteContact struct {
	Heading SiteHeadingTriple `json:"heading"`
	Email   SiteContactEmail  `json:"email"`
	Social  SiteContactSocial `json:"social"`
}

type SiteContactEmail struct {
	Label   string         `json:"label"`
	Address string         `json:"address"`
	CTA     SiteContactCTA `json:"cta"`
}

type SiteContactCTA struct {
	Label        string `json:"label"`
	Variant      string `json:"variant"`
	TrailingIcon string `json:"trailingIcon"`
}

type SiteContactSocial struct {
	Label string            `json:"label"`
	Links []SiteContactLink `json:"links"`
}

type SiteContactLink struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type SiteFooter struct {
	CopyrightTemplate string `json:"copyrightTemplate"`
}

// ParseAndValidateSiteDataJSON decodes JSON with unknown fields rejected, then validates required content.
func ParseAndValidateSiteDataJSON(raw []byte) (SiteDataDocument, error) {
	var zero SiteDataDocument
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return zero, fmt.Errorf("empty body")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var doc SiteDataDocument
	if err := dec.Decode(&doc); err != nil {
		return zero, err
	}
	if err := requireVentureSeedKeys(raw); err != nil {
		return zero, err
	}
	if err := doc.Validate(); err != nil {
		return zero, err
	}
	if dec.More() {
		return zero, fmt.Errorf("trailing JSON after document")
	}
	return doc, nil
}

// Validate ensures every section and list item required by the product schema is present and non-degenerate.
func (d *SiteDataDocument) Validate() error {
	if d == nil {
		return fmt.Errorf("nil document")
	}
	if err := requireNonEmpty("meta.title", d.Meta.Title); err != nil {
		return err
	}
	if err := requireNonEmpty("meta.description", d.Meta.Description); err != nil {
		return err
	}
	if err := requireNonEmpty("meta.siteUrl", d.Meta.SiteURL); err != nil {
		return err
	}
	if err := requireNonEmpty("meta.brandName", d.Meta.BrandName); err != nil {
		return err
	}
	if len(d.Nav.Items) == 0 {
		return fmt.Errorf("nav.items must not be empty")
	}
	for i, it := range d.Nav.Items {
		if err := requireNonEmpty(fmt.Sprintf("nav.items[%d].label", i), it.Label); err != nil {
			return err
		}
		if err := requireNonEmpty(fmt.Sprintf("nav.items[%d].href", i), it.Href); err != nil {
			return err
		}
	}
	if err := requireNonEmpty("hero.pill", d.Hero.Pill); err != nil {
		return err
	}
	if err := requireNonEmpty("hero.name", d.Hero.Name); err != nil {
		return err
	}
	if len(d.Hero.Taglines) == 0 {
		return fmt.Errorf("hero.taglines must not be empty")
	}
	for i, t := range d.Hero.Taglines {
		if err := requireNonEmpty(fmt.Sprintf("hero.taglines[%d]", i), t); err != nil {
			return err
		}
	}
	if d.Hero.TaglineIntervalMs <= 0 {
		return fmt.Errorf("hero.taglineIntervalMs must be positive")
	}
	if len(d.Hero.CTAs) == 0 {
		return fmt.Errorf("hero.ctas must not be empty")
	}
	for i, c := range d.Hero.CTAs {
		p := fmt.Sprintf("hero.ctas[%d]", i)
		if err := requireNonEmpty(p+".id", c.ID); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".label", c.Label); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".variant", c.Variant); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".size", c.Size); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".trailingIcon", c.TrailingIcon); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".action.type", c.Action.Type); err != nil {
			return err
		}
		if c.Action.Type == "scroll" && strings.TrimSpace(c.Action.TargetID) == "" {
			return fmt.Errorf("%s.action.targetId is required for scroll actions", p)
		}
	}
	if err := validateHeadingTriple("ventures.heading", d.Ventures.Heading); err != nil {
		return err
	}
	if err := requireNonEmpty("ventures.ui.statusLabels.live", d.Ventures.UI.StatusLabels.Live); err != nil {
		return err
	}
	if err := requireNonEmpty("ventures.ui.statusLabels.comingSoon", d.Ventures.UI.StatusLabels.ComingSoon); err != nil {
		return err
	}
	if err := requireNonEmpty("ventures.ui.imagePlaceholder", d.Ventures.UI.ImagePlaceholder); err != nil {
		return err
	}
	if err := requireNonEmpty("ventures.ui.carouselAriaLabel", d.Ventures.UI.CarouselAriaLabel); err != nil {
		return err
	}
	if d.Ventures.UI.SkeletonCount < 0 {
		return fmt.Errorf("ventures.ui.skeletonCount must be >= 0")
	}
	if len(d.Ventures.Seed) == 0 {
		return fmt.Errorf("ventures.seed must not be empty")
	}
	for i, v := range d.Ventures.Seed {
		p := fmt.Sprintf("ventures.seed[%d]", i)
		if err := requireNonEmpty(p+".id", v.ID); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".title", v.Title); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".description", v.Description); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".longDescription", v.LongDescription); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".status", v.Status); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".href", v.Href); err != nil {
			return err
		}
	}
	if err := validateHeadingTriple("about.heading", d.About.Heading); err != nil {
		return err
	}
	if len(d.About.Cards) == 0 {
		return fmt.Errorf("about.cards must not be empty")
	}
	for i, c := range d.About.Cards {
		p := fmt.Sprintf("about.cards[%d]", i)
		if err := requireNonEmpty(p+".title", c.Title); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".body", c.Body); err != nil {
			return err
		}
	}
	if err := validateHeadingTriple("skills.heading", d.Skills.Heading); err != nil {
		return err
	}
	if len(d.Skills.Items) == 0 {
		return fmt.Errorf("skills.items must not be empty")
	}
	for i, s := range d.Skills.Items {
		if err := requireNonEmpty(fmt.Sprintf("skills.items[%d]", i), s); err != nil {
			return err
		}
	}
	if d.Skills.Marquee.DurationSeconds <= 0 {
		return fmt.Errorf("skills.marquee.durationSeconds must be positive")
	}
	if err := validateHeadingTriple("contact.heading", d.Contact.Heading); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.email.label", d.Contact.Email.Label); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.email.address", d.Contact.Email.Address); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.email.cta.label", d.Contact.Email.CTA.Label); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.email.cta.variant", d.Contact.Email.CTA.Variant); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.email.cta.trailingIcon", d.Contact.Email.CTA.TrailingIcon); err != nil {
		return err
	}
	if err := requireNonEmpty("contact.social.label", d.Contact.Social.Label); err != nil {
		return err
	}
	if len(d.Contact.Social.Links) == 0 {
		return fmt.Errorf("contact.social.links must not be empty")
	}
	for i, l := range d.Contact.Social.Links {
		p := fmt.Sprintf("contact.social.links[%d]", i)
		if err := requireNonEmpty(p+".label", l.Label); err != nil {
			return err
		}
		if err := requireNonEmpty(p+".href", l.Href); err != nil {
			return err
		}
	}
	if err := requireNonEmpty("footer.copyrightTemplate", d.Footer.CopyrightTemplate); err != nil {
		return err
	}
	return nil
}

func validateHeadingTriple(prefix string, h SiteHeadingTriple) error {
	if err := requireNonEmpty(prefix+".eyebrow", h.Eyebrow); err != nil {
		return err
	}
	if err := requireNonEmpty(prefix+".title", h.Title); err != nil {
		return err
	}
	if err := requireNonEmpty(prefix+".description", h.Description); err != nil {
		return err
	}
	return nil
}

func requireNonEmpty(path, s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("%s is required", path)
	}
	return nil
}

func requireVentureSeedKeys(raw []byte) error {
	var probe struct {
		Ventures struct {
			Seed []map[string]json.RawMessage `json:"seed"`
		} `json:"ventures"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return err
	}
	keys := []string{"id", "title", "description", "longDescription", "status", "imageUrl", "href"}
	for i, row := range probe.Ventures.Seed {
		for _, k := range keys {
			if _, ok := row[k]; !ok {
				return fmt.Errorf("ventures.seed[%d] missing required field %q", i, k)
			}
		}
	}
	return nil
}
