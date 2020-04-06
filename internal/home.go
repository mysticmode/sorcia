package internal

import (
	"database/sql"
	"html/template"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"sorcia/models"
	"sorcia/pkg"
	"strings"
)

// IndexPageResponse struct
type IndexPageResponse struct {
	IsLoggedIn       bool
	ShowLoginMenu    bool
	HeaderActiveMenu string
	SorciaVersion    string
	CanCreateRepo    bool
	Repos            GetReposStruct
	SiteSettings     SiteSettings
}

// GetReposStruct struct
type GetReposStruct struct {
	Repositories []RepoDetailStruct
}

// RepoDetailStruct struct
type RepoDetailStruct struct {
	ID          int
	Name        string
	Description string
	IsPrivate   bool
	Permission  string
}

// GetHome ...
func GetHome(w http.ResponseWriter, r *http.Request, db *sql.DB, conf *pkg.BaseStruct) {
	userPresent := w.Header().Get("user-present")

	repos := models.GetAllPublicRepos(db)
	var grs GetReposStruct

	if userPresent == "true" {
		token := w.Header().Get("sorcia-cookie-token")
		userID := models.GetUserIDFromToken(db, token)

		for _, repo := range repos.Repositories {
			rd := RepoDetailStruct{
				ID:          repo.ID,
				Name:        repo.Name,
				Description: repo.Description,
				IsPrivate:   repo.IsPrivate,
				Permission:  repo.Permission,
			}
			if models.CheckRepoMemberExistFromUserIDAndRepoID(db, userID, repo.ID) {
				rd.Permission = models.GetRepoMemberPermissionFromUserIDAndRepoID(db, userID, repo.ID)
			} else if models.CheckRepoOwnerFromUserIDAndReponame(db, userID, repo.Name) {
				rd.Permission = "read/write"
			}

			grs.Repositories = append(grs.Repositories, rd)
		}

		var reposAsMember models.GetReposStruct
		var repoAsMember models.RepoDetailStruct

		reposAsMember = models.GetReposFromUserID(db, userID)

		repoIDs := models.GetRepoIDsOnRepoMembersUsingUserID(db, userID)
		for _, repoID := range repoIDs {
			repoAsMember = models.GetRepoFromRepoID(db, repoID)
			reposAsMember.Repositories = append(reposAsMember.Repositories, repoAsMember)
		}

		for _, repo := range reposAsMember.Repositories {
			repoExistCount := 0
			for _, publicRepo := range grs.Repositories {
				if publicRepo.Name == repo.Name {
					repoExistCount = 1
				}
			}

			if repoExistCount == 0 {
				rd := RepoDetailStruct{
					ID:          repo.ID,
					Name:        repo.Name,
					Description: repo.Description,
					IsPrivate:   repo.IsPrivate,
					Permission:  repo.Permission,
				}
				if models.CheckRepoMemberExistFromUserIDAndRepoID(db, userID, repo.ID) {
					rd.Permission = models.GetRepoMemberPermissionFromUserIDAndRepoID(db, userID, repo.ID)
				} else if models.CheckRepoOwnerFromUserIDAndReponame(db, userID, repo.Name) {
					rd.Permission = "read/write"
				}

				grs.Repositories = append(grs.Repositories, rd)
			}
		}

		layoutPage := path.Join("./public/templates", "layout.html")
		headerPage := path.Join("./public/templates", "header.html")
		indexPage := path.Join("./public/templates", "index.html")
		footerPage := path.Join("./public/templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		pkg.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsLoggedIn:       true,
			HeaderActiveMenu: "",
			SorciaVersion:    conf.Version,
			CanCreateRepo:    models.CheckifUserCanCreateRepo(db, userID),
			Repos:            grs,
			SiteSettings:     GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	} else {
		if !models.CheckIfFirstUserExists(db) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		for _, repo := range repos.Repositories {
			rd := RepoDetailStruct{
				ID:          repo.ID,
				Name:        repo.Name,
				Description: repo.Description,
				IsPrivate:   repo.IsPrivate,
				Permission:  repo.Permission,
			}
			grs.Repositories = append(grs.Repositories, rd)
		}

		layoutPage := path.Join("./public/templates", "layout.html")
		headerPage := path.Join("./public/templates", "header.html")
		indexPage := path.Join("./public/templates", "index.html")
		footerPage := path.Join("./public/templates", "footer.html")

		tmpl, err := template.ParseFiles(layoutPage, headerPage, indexPage, footerPage)
		pkg.CheckError("Error on template parse", err)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		data := IndexPageResponse{
			IsLoggedIn:    false,
			ShowLoginMenu: true,
			SorciaVersion: conf.Version,
			Repos:         grs,
			SiteSettings:  GetSiteSettings(db, conf),
		}

		tmpl.ExecuteTemplate(w, "layout", data)
	}
}

// SiteSettings struct
type SiteSettings struct {
	IsSiteTitle    bool
	IsSiteFavicon  bool
	IsSiteLogo     bool
	SiteTitle      string
	SiteStyle      string
	SiteFavicon    string
	SiteFaviconExt string
	SiteLogo       string
	SiteLogoWidth  string
	SiteLogoHeight string
	IsSiteLogoSVG  bool
	SVGDAT         template.HTML
}

// GetSiteSettings ...
func GetSiteSettings(db *sql.DB, conf *pkg.BaseStruct) SiteSettings {
	gssr := models.GetSiteSettings(db, pkg.GetConf())

	isSiteTitle := true
	if gssr.Title == "" {
		isSiteTitle = false
	}

	isSiteFavicon := true
	if gssr.Favicon == "" {
		isSiteFavicon = false
	}

	isSiteLogo := true
	if gssr.Logo == "" {
		isSiteLogo = false
	}

	var faviconExt string
	faviconSplit := strings.Split(gssr.Favicon, ".")
	if len(faviconSplit) > 1 {
		faviconExt = faviconSplit[1]
	}

	var isSiteLogoSVG bool
	var svgXML template.HTML
	var siteLogoExt string
	siteLogoSplit := strings.Split(gssr.Logo, ".")
	if len(siteLogoSplit) > 1 {
		siteLogoExt = siteLogoSplit[1]
	}
	if siteLogoExt == "svg" {
		isSiteLogoSVG = true
		dat, err := ioutil.ReadFile(filepath.Join(conf.Paths.UploadAssetPath, gssr.Logo))
		pkg.CheckError("Error on Reading svg logo file", err)

		svgXML = template.HTML(dat)
	}

	siteSettings := SiteSettings{
		IsSiteTitle:    isSiteTitle,
		IsSiteFavicon:  isSiteFavicon,
		IsSiteLogo:     isSiteLogo,
		SiteTitle:      gssr.Title,
		SiteStyle:      gssr.Style,
		SiteFavicon:    gssr.Favicon,
		SiteFaviconExt: faviconExt,
		SiteLogo:       gssr.Logo,
		SiteLogoWidth:  gssr.LogoWidth,
		SiteLogoHeight: gssr.LogoHeight,
		IsSiteLogoSVG:  isSiteLogoSVG,
		SVGDAT:         svgXML,
	}

	return siteSettings
}
