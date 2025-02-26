package models

import (
	ipfs "dreamfriday/IPFS"
	database "dreamfriday/database"
	"log"
)

func GetSite(name string) (*Site, error) {
	var site Site
	err := database.Get("Sites", name, &site)
	if err != nil {
		return nil, err
	}
	return &site, nil
}

// @TODO make sure caller sets published status
func UpdateSite(name string, site *Site) error {
	return database.Put("Sites", name, site)
}

func CreateSite(name, owner, siteData string) (*Site, error) {
	hash, err := ipfs.PutFile(siteData)
	if err != nil {
		log.Printf("Failed to add site data for %s on ipfs: %v", name, err)
		return nil, err
	}
	log.Printf("Saved site %s on ipfs: %s", name, hash)
	site := Site{
		IPFSHash:    hash,
		PreviewData: siteData,
		Owner:       owner,
		Status:      "published",
	}
	err = database.Put("Sites", name, site)
	if err != nil {
		log.Printf("Failed to create site for %s: %v", name, err)
		return nil, err
	}
	log.Println("Added site to bolt:", name)
	return &site, nil
}

// func UpdatePreviewData(name, previewData string) error {
// 	site, err := GetSite(name)
// 	if err != nil {
// 		return err
// 	}
// 	site.PreviewData = previewData
// 	return UpdateSite(name, site)
// }

func PublishSite(site *Site) error {

	previewData := site.PreviewData
	hash := site.IPFSHash
	siteName := site.Name

	log.Println("Publishing site:", siteName)
	log.Println("TODO: Unpin hash:", hash)

	hash, err := ipfs.PutFile(previewData)
	if err != nil {
		log.Printf("Failed to save site to ipfs %s: %v", siteName, err)
		return err
	}
	log.Printf("Saved site %s on ipfs: %s", siteName, hash)
	site.IPFSHash = hash
	site.Status = "published"
	err = UpdateSite(siteName, site)
	if err != nil {
		log.Printf("Failed to update site %s: %v", siteName, err)
	}

	log.Println("TODO: Pin hash:", hash)

	return nil
}

func GetSiteData(name string) (string, error) {
	log.Println("Getting site data for:", name)
	site, err := GetSite(name)
	if err != nil {
		return "", err
	}
	ipfsHash := site.IPFSHash
	data, err := ipfs.GetFile(ipfsHash)
	if err != nil {
		log.Printf("Failed to get site data for %s: %v", name, err)
		return "", err
	}
	log.Printf("Retrieved site data for %s (%s)", name, ipfsHash)
	return data, nil
}
