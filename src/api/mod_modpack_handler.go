package api

import (
	"archive/zip"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mroote/factorio-server-manager/bootstrap"
	"github.com/mroote/factorio-server-manager/factorio"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func CheckModPackExists(modPackMap factorio.ModPackMap, modPackName string, w http.ResponseWriter, resp interface{}) error {
	exists := modPackMap.CheckModPackExists(modPackName)
	if !exists {
		resp = fmt.Sprintf("requested modPack {%s} does not exist", modPackName)
		log.Println(resp)
		w.WriteHeader(http.StatusNotFound)
		return errors.New("requested modPack does not exist")
	}
	return nil
}

func CreateNewModPackMap(w http.ResponseWriter, resp *interface{}) (modPackMap factorio.ModPackMap, err error) {
	modPackMap, err = factorio.NewModPackMap()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		*resp = fmt.Sprintf("Error creating modpackmap aka. list of all modpacks files : %s", err)
		log.Println(*resp)
	}
	return
}

func ReadModPackRequest(w http.ResponseWriter, r *http.Request, resp *interface{}) (err error, packMap factorio.ModPackMap, modPackName string) {
	vars := mux.Vars(r)
	modPackName = vars["modpack"]

	packMap, err = CreateNewModPackMap(w, resp)
	if err != nil {
		return
	}

	if err = CheckModPackExists(packMap, modPackName, w, resp); err != nil {
		return
	}
	return
}

//////////////////////
// Mod Pack Handler //
//////////////////////

func ModPackListHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	modPackMap, err := CreateNewModPackMap(w, &resp)
	if err != nil {
		return
	}

	resp = modPackMap.ListInstalledModPacks()
}

func ModPackCreateHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var modPackStruct struct {
		Name string `json:"name"`
	}
	err = ReadFromRequestBody(w, r, &resp, &modPackStruct)
	if err != nil {
		return
	}

	modPackMap, err := CreateNewModPackMap(w, &resp)
	if err != nil {
		return
	}

	err = modPackMap.CreateModPack(modPackStruct.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error creating modpack file: %s", err)
		log.Println(resp)
		return
	}

	resp = modPackMap.ListInstalledModPacks()
}

func ModPackDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	err, modPackMap, modPackName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	err = modPackMap.DeleteModPack(modPackName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error deleting modpack file: %s", err)
		log.Println(resp)
		return
	}

	resp = modPackName
}

func ModPackDownloadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	err, _, modPackName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		WriteResponse(w, resp)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", modPackName))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	config := bootstrap.GetConfig()

	//iterate over folder and create everything in the zip
	err = filepath.Walk(filepath.Join(config.FactorioModPackDir, modPackName), func(path string, info os.FileInfo, err error) error {
		if info.IsDir() == false {
			writer, err := zipWriter.Create(info.Name())
			if err != nil {
				log.Printf("error on creating new file inside zip: %s", err)
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				log.Printf("error on opening modfile: %s", err)
				return err
			}
			// Close file, when function returns
			defer func() {
				err2 := file.Close()
				if err == nil && err2 != nil {
					log.Printf("Error closing file: %s", err2)
					err = err2
				}
			}()

			_, err = io.Copy(writer, file)
			if err != nil {
				log.Printf("error on copying file into zip: %s", err)
				return err
			}
		}

		return nil
	})
	if err != nil {
		resp = fmt.Sprintf("error on walking over the modpack: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json;charset=UTF-8")
		WriteResponse(w, resp)
		return
	}

	w.Header().Set("Content-Type", "application/zip;charset=UTF-8")
}

func ModPackLoadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	err, modPackMap, modPackName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	err = modPackMap[modPackName].LoadModPack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error loading modpack file: %s", err)
		log.Println(resp)
		return
	}

	resp = modPackMap[modPackName].Mods.ListInstalledMods()
}

//////////////////////////////////
// Mods inside Mod Pack Handler //
//////////////////////////////////
func ModPackModListHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	err, modPackMap, modPackName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	resp = modPackMap[modPackName].Mods.ListInstalledMods()
}

func ModPackModToggleHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	var modPackStruct struct {
		ModName string `json:"name"`
	}
	ReadFromRequestBody(w, r, &resp, &modPackStruct)
	if err != nil {
		return
	}

	err, resp = packMap[packName].Mods.ModSimpleList.ToggleMod(modPackStruct.ModName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error toggling mod inside modPack: %s", err)
		log.Println(resp)
		return
	}
}

func ModPackModDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	var modPackStruct struct {
		Name string `json:"name"`
	}
	err = ReadFromRequestBody(w, r, &resp, &modPackStruct)
	if err != nil {
		return
	}

	err = packMap[packName].Mods.DeleteMod(modPackStruct.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error deleting mod {%s} in modpack {%s}: %s", modPackStruct.Name, packName, err)
		log.Println(resp)
		return
	}

	resp = true
}

func ModPackModUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	//Get Data out of the request
	var modPackStruct struct {
		ModName     string `json:"modName"`
		DownloadUrl string `json:"downloadUrl"`
		Filename    string `json:"filename"`
	}
	err = ReadFromRequestBody(w, r, &resp, &modPackStruct)
	if err != nil {
		return
	}

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	err = packMap[packName].Mods.UpdateMod(modPackStruct.ModName, modPackStruct.DownloadUrl, modPackStruct.Filename)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		resp = fmt.Sprintf("Error updating mod {%s} in modpack {%s}: %s", modPackStruct.ModName, packName, err)
		log.Println(resp)
		return
	}

	installedMods := packMap[packName].Mods.ListInstalledMods().ModsResult
	var found = false
	for _, mod := range installedMods {
		if mod.Name == modPackStruct.ModName {
			resp = mod
			found = true
			return
		}
	}

	if !found {
		resp = fmt.Sprintf(`Could not find mod %s`, modPackStruct.ModName)
		log.Println(resp)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func ModPackModDeleteAllHandler(w http.ResponseWriter, r *http.Request) {
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	// Delete Modpack
	err = packMap.DeleteModPack(packName)
	if err != nil {
		resp = fmt.Sprintf("Error deleting modPackDir: %s", err)
		log.Println(resp)
		return
	}

	// recreate modPack without mods
	err = packMap.CreateEmptyModPack(packName)
	if err != nil {
		resp = fmt.Sprintf("Error recreating modPackDir: %s", err)
		log.Println(resp)
		return
	}

	resp = true
}

func ModPackModUploadHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	formFile, fileHeader, err := r.FormFile("mod_file")
	if err != nil {
		resp = fmt.Sprintf("error getting uploaded file: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer formFile.Close()

	err, modPackMap, modPackName := ReadModPackRequest(w, r, &resp)

	err = modPackMap[modPackName].Mods.UploadMod(formFile, fileHeader)
	if err != nil {
		resp = fmt.Sprintf("error saving file to modPack: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = modPackMap[modPackName].Mods.ListInstalledMods()
}

func ModPackModPortalInstallHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	// Get Data out of the request
	var data struct {
		DownloadURL string `json:"downloadUrl"`
		Filename    string `json:"fileName"`
		ModName     string `json:"modName"`
	}
	err = ReadFromRequestBody(w, r, &resp, &data)
	if err != nil {
		return
	}

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}

	modList := packMap[packName].Mods

	err = modList.DownloadMod(data.DownloadURL, data.Filename, data.ModName)
	if err != nil {
		resp = fmt.Sprintf("Error downloading a mod: %s", err)
		log.Println(resp)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp = modList.ListInstalledMods()
}

func ModPackModPortalInstallMultipleHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var resp interface{}

	defer func() {
		WriteResponse(w, resp)
	}()

	w.Header().Set("Content-Type", "application/json;charset=UTF-8")

	var data []struct {
		Name    string           `json:"name"`
		Version factorio.Version `json:"version"`
	}
	err = ReadFromRequestBody(w, r, &resp, &data)
	if err != nil {
		return
	}

	err, packMap, packName := ReadModPackRequest(w, r, &resp)
	if err != nil {
		return
	}
	modList := packMap[packName].Mods
	for _, datum := range data {
		details, err, statusCode := factorio.ModPortalModDetails(datum.Name)
		if err != nil || statusCode != http.StatusOK {
			resp = fmt.Sprintf("Error in getting mod details from mod portal: %s", err)
			log.Println(resp)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//find correct mod-version
		var found = false

		for _, release := range details.Releases {
			if release.Version.Equals(datum.Version) {
				found = true

				err := modList.DownloadMod(release.DownloadURL, release.FileName, details.Name)
				if err != nil {
					resp = fmt.Sprintf("Error downloading mod {%s}, error: %s", details.Name, err)
					log.Println(resp)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				break
			}
		}

		if !found {
			log.Printf("Error downloading mod {%s}, error: %s", details.Name, "version not found")
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	resp = modList.ListInstalledMods()
}
