package pcpartpicker_automation

import (
	"errors"
	"fmt"
	"github.com/Aquilabot/KreaPC-API/internal/models"
	"github.com/Aquilabot/KreaPC-API/internal/utils"
	"github.com/gofiber/fiber/v2/log"
	"github.com/playwright-community/playwright-go"
)

const (
	errorInvalidRegion          = "invalid region"
	errorInitializingPlaywright = "could not start Playwright: %v"
	errorLaunchingBrowser       = "could not launch browser: %v"
	errorCreatingPage           = "could not create page: %v"
	errorNavigatingURL          = "could not navigate to %s: %v"
	logInitPlaywright           = "Initializing Playwright"
	logErrorCookies             = "Error handling cookies, but we continue: %v"
	logCleanupPlaywright        = "Cleaning up Playwright"
	logErrorCloseBrowser        = "Could not close browser: %v"
	logErrorStopPlaywright      = "Could not stop Playwright: %v"
)

func ProcessPartLinks(region string, partLinks []string) (*models.SearchPart, error) {
	prefixURL := utils.BuildPrefixURL(region)
	if !utils.MatchPCPPURL(prefixURL) {
		return nil, errors.New(errorInvalidRegion)
	}
	pw, browser, page, err := initializePlaywright()
	if err != nil {
		return nil, err
	}
	defer cleanup(pw, browser)

	if err := navigateTo(page, prefixURL); err != nil {
		return nil, err
	}

	if err := handleCookies(page); err != nil {
		log.Warnf(logErrorCookies, err)
	}

	if err := addPartsList(prefixURL, page, partLinks); err != nil {
		return nil, err
	}

	return handleTextbox(page)
}

func initializePlaywright() (*playwright.Playwright, playwright.Browser, playwright.Page, error) {
	log.Info(logInitPlaywright)
	pw, err := playwright.Run()
	if err != nil {
		return nil, nil, nil, fmt.Errorf(errorInitializingPlaywright, err)
	}
	browser, err := pw.Chromium.Launch()
	if err != nil {
		return nil, nil, nil, fmt.Errorf(errorLaunchingBrowser, err)
	}
	page, err := browser.NewPage()
	if err != nil {
		return nil, nil, nil, fmt.Errorf(errorCreatingPage, err)
	}
	return pw, browser, page, nil
}

func navigateTo(page playwright.Page, url string) error {
	if _, err := page.Goto(url); err != nil {
		return fmt.Errorf(errorNavigatingURL, url, err)
	}
	return nil
}

func handleCookies(page playwright.Page) error {
	return page.GetByLabel("allow cookies").Click()
}

func addPart(prefixURL string, page playwright.Page, url string) error {
	if err := navigateTo(page, url); err != nil {
		return err
	}
	options := playwright.PageGetByRoleOptions{Name: "Add to Part List"}
	if err := page.GetByRole("link", options).Click(); err != nil {
		return fmt.Errorf("could not click 'Add to Part List': %v", err)
	}

	if _, err := page.ExpectNavigation(func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("error waiting for navigation to start: %v", err)
	}

	if err := page.WaitForURL(prefixURL + "list/"); err != nil {
		return fmt.Errorf("error waiting for redirection to the list: %v", err)
	}
	log.Info("Added to Part List and redirection complete")
	return nil
}

func addPartsList(prefixURL string, page playwright.Page, links []string) error {
	for _, link := range links {
		if err := addPart(prefixURL, page, link); err != nil {
			return fmt.Errorf("error adding part from link %s: %v\n", link, err)
		}
	}
	return nil
}

func handleTextbox(page playwright.Page) (*models.SearchPart, error) {
	textboxLocator := page.GetByRole("textbox")
	if err := textboxLocator.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateAttached}); err != nil {
		return nil, err
	}

	if err := textboxLocator.WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible}); err != nil {
		return nil, fmt.Errorf("could not wait for the textbox to be visible: %v", err)
	}

	value, err := textboxLocator.InputValue()
	if err != nil {
		return nil, err
	}

	return &models.SearchPart{
		URL: value,
	}, nil
}

func cleanup(pw *playwright.Playwright, browser playwright.Browser) {
	log.Info(logCleanupPlaywright)
	if err := browser.Close(); err != nil {
		log.Fatalf(logErrorCloseBrowser, err)
	}
	if err := pw.Stop(); err != nil {
		log.Fatalf(logErrorStopPlaywright, err)
	}
}
