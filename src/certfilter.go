package main

/*
	certfilter.go

	Filtrovaný výpis TLS / SSL certifikátů. Uživatel vidí jen certifikáty pro
	domény, ke kterým má oprávnění (stejná logika jako u HTTP proxy domén).
	Admin a uživatelé s globálním oprávněním (domainsAll) vidí všechny certy.
*/

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"imuslab.com/zoraxy/mod/acme"
	"imuslab.com/zoraxy/mod/utils"
)

func handleCertListFiltered(w http.ResponseWriter, r *http.Request) {
	username, _ := authAgent.GetUsernameFromRequest(r)

	//Admin / globální přístup: vrať vše beze změny.
	if authAgent.IsAdmin(r) || authAgent.GetUserDomainsAll(username) {
		tlsCertManager.HandleListCertificate(w, r)
		return
	}

	filenames, err := tlsCertManager.ListCertDomains()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	showDate, _ := utils.GetBool(r, "date")
	if showDate {
		type CertInfo struct {
			Domain           string
			Filename         string
			LastModifiedDate string
			ExpireDate       string
			RemainingDays    int
			UseDNS           bool
			IsFallback       bool
		}

		results := []*CertInfo{}
		for _, filename := range filenames {
			//Uživatel vidí jen certifikát pro doménu, ke které má přístup.
			if !authAgent.UserCanAccessDomain(username, filename) {
				continue
			}

			certFilepath := filepath.Join(tlsCertManager.CertStore, filename+".pem")
			fileInfo, err := os.Stat(certFilepath)
			if err != nil {
				continue
			}
			modifiedTime := fileInfo.ModTime().Format("2006-01-02 15:04:05")

			certExpireTime := "Unknown"
			expiredIn := 0
			certBytes, err := os.ReadFile(certFilepath)
			if err != nil {
				continue
			}

			certDomain := ""
			block, _ := pem.Decode(certBytes)
			if block != nil {
				cert, err := x509.ParseCertificate(block.Bytes)
				if err == nil {
					certExpireTime = cert.NotAfter.Format("2006-01-02 15:04:05")
					duration := time.Until(cert.NotAfter)
					expiredIn = int(duration.Hours() / 24)
					certDomain = cert.Subject.CommonName
				}
			}

			certInfoFilename := filepath.Join(tlsCertManager.CertStore, filename+".json")
			useDNSValidation := false
			certInfo, err := acme.LoadCertInfoJSON(certInfoFilename)
			if err == nil {
				useDNSValidation = certInfo.UseDNS
			}

			results = append(results, &CertInfo{
				Domain:           certDomain,
				Filename:         filename,
				LastModifiedDate: modifiedTime,
				ExpireDate:       certExpireTime,
				RemainingDays:    expiredIn,
				UseDNS:           useDNSValidation,
				IsFallback:       (filename == tlsCertManager.FallbackCert),
			})
		}

		sort.Slice(results, func(i, j int) bool {
			date1, _ := time.Parse("2006-01-02 15:04:05", results[i].ExpireDate)
			date2, _ := time.Parse("2006-01-02 15:04:05", results[j].ExpireDate)
			return date1.Before(date2)
		})

		js, _ := json.Marshal(results)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	//date=false: vrať jen povolené názvy certifikátů.
	filtered := []string{}
	for _, filename := range filenames {
		if authAgent.UserCanAccessDomain(username, filename) {
			filtered = append(filtered, filename)
		}
	}
	js, _ := json.Marshal(filtered)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
