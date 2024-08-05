package scraper

import (
	"errors"
	"github.com/Aquilabot/KreaPC-API/internal/models"
	"github.com/Aquilabot/KreaPC-API/internal/utils"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gofiber/fiber/v2/log"
	"net/url"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

var (
	partListClassMappings = map[string]string{
		"Base":     ".td__base",
		"Promo":    ".td__promo",
		"Shipping": ".td__shipping",
		"Tax":      ".td__tax",
		"Price":    ".td__price",
	}

	partClassMappings = map[string]string{
		"Base":     ".td__base",
		"Promo":    ".td__promo",
		"Shipping": ".td__shipping",
		"Tax":      ".td__tax",
		"Total":    ".td__finalPrice",
	}
)

type Scraper struct {
	Collector *colly.Collector
	Headers   map[string]map[string]string
}

type RedirectError struct {
	URL string
}

func (r RedirectError) Error() string {
	return r.URL
}

func linkURL(parts ...string) string {
	last := parts[len(parts)-1]
	if last == "" {
		return ""
	} else if strings.HasPrefix(last, "http") {
		return last
	}
	return strings.Join(parts, "")
}

func buildPCPartPickerURL(searchTerm string, region string) string {
	if region != "" && region != "us" {
		region += "."
	} else {
		region = ""
	}

	fullURL := "https://" + region + "pcpartpicker.com/search?q=" + url.QueryEscape(searchTerm)
	return fullURL
}

// NewScraper initializes a new instance of the Scraper type and returns it.
// It creates a new collector, sets the async and AllowURLRevisit properties to true,
// and initializes an empty Headers map with the "global" site.
func NewScraper() Scraper {
	col := colly.NewCollector()
	col.Async = true
	col.AllowURLRevisit = true

	s := Scraper{
		Collector: col,
	}
	s.Headers = map[string]map[string]string{
		"global": {},
	}

	return s
}

// UpdateHeaders updates the headers for the given site with the provided newHeaders map.
// It updates the headers for the "global" site as well.
// It also sets the headers for each request made by the Collector.
func (scrap *Scraper) UpdateHeaders(site string, newHeaders map[string]string) {
	scrap.Headers[site] = newHeaders

	for k, v := range newHeaders {
		scrap.Headers[site][k] = v
	}

	scrap.Collector.OnRequest(func(r *colly.Request) {
		headers := scrap.Headers["global"]
		for k, v := range scrap.Headers[r.URL.Hostname()] {
			headers[k] = v
		}

		for k, v := range headers {
			if len(k) > 0 && len(v) > 0 {
				r.Headers.Set(k, v)
			}
		}
	})
}

func (scrap *Scraper) RandomizeUserAgent() {
	extensions.RandomUserAgent(scrap.Collector)
	scrap.Collector.OnRequest(func(r *colly.Request) {
		log.Info("User-Agent:", r.Headers.Get("User-Agent"))
	})
}

// GetPartList retrieves a list of parts from the given PCPartPicker URL.
// It returns a pointer to models.PartList and an error.
// If the URL is invalid, it returns an error.
func (scrap *Scraper) GetPartList(URL string) (*models.PartList, error) {
	if !utils.MatchPCPPURL(URL) {
		return nil, errors.New("invalid PCPartPicker URL")
	}
	URL = utils.ConvertListURL(URL)

	var partList models.PartList

	scrap.Collector.OnHTML(".partlist__wrapper", func(elem *colly.HTMLElement) {
		parts := []models.ListPart{}

		elem.ForEach(".tr__product", func(i int, prod *colly.HTMLElement) {
			prodVendor := models.Vendor{
				InStock: false,
				Price:   models.Price{},
			}

			for k, v := range partListClassMappings {
				toParse := prod.ChildText(v)

				if strings.HasSuffix(toParse, "No Prices Available") || toParse == "FREE" {
					continue
				}
				stringPrice := strings.Replace(toParse, k, "", 1)
				price, curr, _ := models.ParsePrice(stringPrice)

				switch k {
				case "Base":
					prodVendor.Price.Base = price
				case "Promo":
					prodVendor.Price.Discounts = price
				case "Shipping":
					prodVendor.Price.Shipping = price
				case "Tax":
					prodVendor.Price.Shipping = price
				case "Price":
					prodVendor.Price.TotalString = strings.TrimSpace(stringPrice)
					prodVendor.Price.Total = price
					prodVendor.Price.Currency = curr
					prodVendor.InStock = true
				}
			}

			if prodVendor.InStock {
				prodVendor.URL = linkURL("https://", elem.Request.URL.Host, prod.ChildAttr(".td__where a", "href"))
				prodVendor.Image = linkURL("https:", prod.ChildAttr(".td__where a img", "src"))
				prodVendor.Name = utils.ExtractVendorName(prodVendor.URL)
			}

			part := models.ListPart{
				Type:   prod.ChildText(".td__component"),
				Name:   prod.ChildText(".td__name"),
				Image:  linkURL("https:", prod.ChildAttr(".td__image a img", "src")),
				URL:    linkURL("https://", elem.Request.URL.Host, prod.ChildAttr(".td__name a", "href")),
				Vendor: prodVendor,
			}

			parts = append(parts, part)
		})

		listPrice := models.Price{}

		elem.ForEach(".tr__total", func(i int, node *colly.HTMLElement) {
			stringPrice := node.ChildText(".td__price")
			val, curr, _ := models.ParsePrice(stringPrice)

			switch node.ChildText(".td__label") {
			case "Base Total:":
				listPrice.Base = val
			case "Tax:":
				listPrice.Tax = val
			case "Promo Discounts:":
				listPrice.Discounts = val
			case "Shipping:":
				listPrice.Shipping = val
			case "Total:":
				listPrice.Total = val
				listPrice.TotalString = stringPrice
				listPrice.Currency = curr
			}
		})

		compNotes := []models.CompatibilityInfo{}

		elem.ForEach("#compatibility_notes .info-message", func(i int, note *colly.HTMLElement) {
			mode := note.ChildText("span")
			compNotes = append(compNotes, models.CompatibilityInfo{
				Message: strings.TrimLeft(strings.TrimSpace(note.Text), mode),
				Level:   strings.TrimRight(mode, ":"),
			})
		})

		partList = models.PartList{
			URL:           elem.Request.URL.String(),
			Parts:         parts,
			Price:         listPrice,
			Wattage:       strings.TrimPrefix(elem.ChildText(".partlist__keyMetric"), "Estimated Wattage:\n"),
			Compatibility: compNotes,
		}
	})
	err := scrap.Collector.Visit(URL)
	scrap.Collector.Wait()

	if err != nil {
		return nil, err
	}

	return &partList, nil
}

// SearchPCParts retrieves a list of parts from the given search term and region.
// It returns a slice of models.SearchPart and an error.
// If the region is invalid, it returns an error.
func (scrap *Scraper) SearchPCParts(searchTerm string, region string) ([]models.SearchPart, error) {
	fullURL := buildPCPartPickerURL(searchTerm, region)

	if !utils.MatchPCPPURL(fullURL) {
		return nil, errors.New("invalid region")
	}

	searchResults := []models.SearchPart{}

	var reqURL string

	scrap.Collector.OnHTML(".pageTitle", func(h *colly.HTMLElement) {
		reqURL = h.Request.URL.String()
	})

	scrap.Collector.OnHTML(".search-results__pageContent .block", func(elem *colly.HTMLElement) {
		elem.ForEach(".list-unstyled li", func(i int, searchResult *colly.HTMLElement) {
			searchResultURL := linkURL("https://", elem.Request.URL.Host, searchResult.ChildAttr(".search_results--price a", "href"))
			extractedPrice := searchResult.ChildText(".search_results--price a")

			price, curr, _ := models.ParsePrice(extractedPrice)

			extractedVendorName := ""

			if extractedPrice != "" {
				extractedVendorName = utils.ExtractVendorName(searchResultURL)
			}

			partVendor := models.Vendor{
				URL:  searchResultURL,
				Name: extractedVendorName,
				Price: models.Price{
					Total:       price,
					TotalString: extractedPrice,
					Currency:    curr,
				},
				InStock: len(extractedPrice) > 0,
			}

			searchResults = append(searchResults, models.SearchPart{
				Name:   searchResult.ChildText(".search_results--link a"),
				Image:  linkURL("https:", searchResult.ChildAttr(".search_results--img a img", "src")),
				URL:    linkURL("https://", elem.Request.URL.Host, searchResult.ChildAttr(".search_results--link a", "href")),
				Vendor: partVendor,
			})
		})
	})

	err := scrap.Collector.Visit(fullURL)
	scrap.Collector.Wait()

	if err != nil {
		return nil, err
	}

	if utils.MatchProductURL(reqURL) {
		return nil, &RedirectError{
			URL: reqURL,
		}
	}

	return searchResults, nil
}

// GetPart retrieves information about a specific part from the given URL.
// It returns a pointer to models.Part and an error.
// If the URL is invalid, it returns an error.
func (scrap *Scraper) GetPart(URL string) (*models.Part, error) {
	if !utils.MatchProductURL(URL) {
		return nil, errors.New("invalid part URL")
	}

	var images []string

	scrap.Collector.OnHTML(".single_image_gallery_box", func(image *colly.HTMLElement) {
		images = append(images, linkURL("https:", image.ChildAttr("a img", "src")))
	})

	if len(images) < 1 {
		scrap.Collector.OnHTML("script", func(script *colly.HTMLElement) {
			images = utils.FindScriptImages(script, images)
		})
	}

	rating := models.RatingStats{}
	var name string

	scrap.Collector.OnHTML(".wrapper__pageTitle section.xs-col-11", func(ratingContainer *colly.HTMLElement) {
		var stars uint
		ratingContainer.ForEach(".product--rating li", func(i int, _ *colly.HTMLElement) {
			stars += 1
		})

		rating.Stars = stars
		name = ratingContainer.ChildText(".pageTitle")

		splitParts := strings.Split(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(ratingContainer.Text, name, ""), ratingContainer.ChildText(".breadcrumb"), "")), ",")

		if len(splitParts) < 2 {
			return
		}

		countParse, _ := strconv.Atoi(strings.Trim(strings.ReplaceAll(splitParts[0], "Ratings", ""), "( "))
		rating.Count = uint(countParse)

		averageParse, _ := strconv.ParseFloat(strings.Trim(strings.ReplaceAll(splitParts[1], "Average", ""), ") "), 64)
		rating.Average = float64(averageParse)
	})

	var vendors []models.Vendor

	scrap.Collector.OnHTML("#prices table tbody tr", func(vendor *colly.HTMLElement) {
		if vendor.Attr("class") != "" {
			return
		}

		price := models.Price{}

		for k, v := range partClassMappings {
			stringPrice := vendor.ChildText(v)
			val, curr, _ := models.ParsePrice(stringPrice)

			switch k {
			case "Base":
				price.Base = val
			case "Shipping":
				price.Shipping = val
			case "Tax":
				price.Tax = val
			case "Discounts":
				price.Discounts = val
			case "Total":
				price.Total = val
				price.Currency = curr
				price.TotalString = stringPrice
			}
		}

		vendors = append(vendors, models.Vendor{
			Name:    vendor.ChildAttr(".td__logo a img", "alt"),
			Image:   linkURL("https:", vendor.ChildAttr(".td__logo a img", "src")),
			InStock: vendor.ChildText(".td__availability") == "In stock",
			URL:     linkURL("https://", vendor.Request.URL.Host, vendor.ChildAttr(".td__finalPrice a", "href")),
			Price:   price,
		})
	})

	var specs []models.PartSpec

	scrap.Collector.OnHTML(".specs", func(specsContainer *colly.HTMLElement) {
		if len(specs) > 0 {
			return
		}
		specsContainer.ForEach(".group", func(i int, spec *colly.HTMLElement) {
			var values []string

			spec.ForEach(".group__content li", func(i int, specValue *colly.HTMLElement) {
				values = append(values, specValue.Text)
			})

			if len(values) == 0 {
				values = []string{spec.ChildText(".group__content")}
			}

			specs = append(specs, models.PartSpec{
				Name:   spec.ChildText(".group__title"),
				Values: values,
			})
		})

	})

	err := scrap.Collector.Visit(URL)
	scrap.Collector.Wait()

	if err != nil {
		return nil, err
	}

	return &models.Part{
		Name:    name,
		Rating:  rating,
		Specs:   specs,
		Vendors: vendors,
		Images:  images,
	}, nil
}
