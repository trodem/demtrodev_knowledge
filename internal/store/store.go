package store

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PackFile struct {
	SchemaVersion int                `json:"schema_version"`
	Description string             `json:"description"`
	Summary     string             `json:"summary"`
	Owner       string             `json:"owner"`
	Tags        []string           `json:"tags"`
	Examples    []string           `json:"examples"`
	Jump        map[string]string  `json:"jump"`
	Run         map[string]string  `json:"run"`
	Projects    map[string]Project `json:"projects"`
	Search      SearchConfig       `json:"search"`
}

type Project struct {
	Path     string            `json:"path"`
	Commands map[string]string `json:"commands"`
}

type SearchConfig struct {
	Knowledge string `json:"knowledge"`
}

func CreatePack(baseDir, name string) error {
	packDir := filepath.Join(baseDir, "packs", name)
	knowledgeDir := filepath.Join(packDir, "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0755); err != nil {
		return err
	}
	packPath := filepath.Join(packDir, "pack.json")
	pf := PackFile{
		SchemaVersion: 1,
		Description: "Pack " + name,
		Summary:     "Commands and knowledge for " + name,
		Examples: []string{
			"dm -p " + name + " find <query>",
			"dm -p " + name + " run <alias>",
		},
		Jump:     map[string]string{},
		Run:      map[string]string{},
		Projects: map[string]Project{},
		Search: SearchConfig{
			Knowledge: filepath.Join("packs", name, "knowledge"),
		},
	}
	if err := writeJSON(packPath, pf); err != nil {
		return err
	}
	return nil
}

func ListPacks(baseDir string) ([]string, error) {
	packsDir := filepath.Join(baseDir, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		packPath := filepath.Join(packsDir, e.Name(), "pack.json")
		if _, err := os.Stat(packPath); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

type PackInfo struct {
	Name        string
	Path        string
	Description string
	Summary     string
	Owner       string
	Tags        []string
	Examples    []string
	Knowledge   string
	Jumps       int
	Runs        int
	Projects    int
	Actions     int
}

func GetPackInfo(baseDir, name string) (PackInfo, error) {
	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	pf, err := LoadPackFile(packPath)
	if err != nil {
		return PackInfo{}, err
	}
	info := PackInfo{
		Name:        name,
		Path:        packPath,
		Description: pf.Description,
		Summary:     pf.Summary,
		Owner:       pf.Owner,
		Tags:        append([]string{}, pf.Tags...),
		Examples:    append([]string{}, pf.Examples...),
		Knowledge:   pf.Search.Knowledge,
		Jumps:       len(pf.Jump),
		Runs:        len(pf.Run),
		Projects:    len(pf.Projects),
		Actions:     countActions(pf.Projects),
	}
	return info, nil
}

func countActions(projects map[string]Project) int {
	total := 0
	for _, p := range projects {
		total += len(p.Commands)
	}
	return total
}

func PackExists(baseDir, name string) bool {
	packPath := filepath.Join(baseDir, "packs", name, "pack.json")
	_, err := os.Stat(packPath)
	return err == nil
}

func ClonePack(baseDir, src, dst string) error {
	if strings.TrimSpace(src) == "" || strings.TrimSpace(dst) == "" {
		return fmt.Errorf("source and destination pack names are required")
	}
	if !PackExists(baseDir, src) {
		return fmt.Errorf("source pack not found: %s", src)
	}
	if PackExists(baseDir, dst) {
		return fmt.Errorf("destination pack already exists: %s", dst)
	}

	srcDir := filepath.Join(baseDir, "packs", src)
	dstDir := filepath.Join(baseDir, "packs", dst)
	if err := copyDirRecursive(srcDir, dstDir); err != nil {
		return err
	}

	dstPackPath := filepath.Join(dstDir, "pack.json")
	pf, err := LoadPackFile(dstPackPath)
	if err != nil {
		return err
	}
	pf.SchemaVersion = 1
	if strings.TrimSpace(pf.Description) == "" || pf.Description == "Pack "+src {
		pf.Description = "Pack " + dst
	}
	if strings.TrimSpace(pf.Summary) == "" || pf.Summary == "Commands and knowledge for "+src {
		pf.Summary = "Commands and knowledge for " + dst
	}
	srcKnowledge := filepath.Clean(filepath.Join("packs", src, "knowledge"))
	currentKnowledge := filepath.Clean(filepath.FromSlash(strings.TrimSpace(pf.Search.Knowledge)))
	if currentKnowledge == srcKnowledge {
		pf.Search.Knowledge = filepath.Join("packs", dst, "knowledge")
	}
	return SavePackFile(dstPackPath, pf)
}

func ActivePackPath(baseDir string) string {
	return filepath.Join(baseDir, ".dm.active-pack")
}

func GetActivePack(baseDir string) (string, error) {
	path := ActivePackPath(baseDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func SetActivePack(baseDir, name string) error {
	path := ActivePackPath(baseDir)
	return os.WriteFile(path, []byte(name+"\n"), 0644)
}

func ClearActivePack(baseDir string) error {
	path := ActivePackPath(baseDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func LoadPackFile(path string) (PackFile, error) {
	var pf PackFile
	if err := readJSON(path, &pf); err != nil {
		if os.IsNotExist(err) {
			return PackFile{
				SchemaVersion: 1,
				Jump:     map[string]string{},
				Run:      map[string]string{},
				Projects: map[string]Project{},
			}, nil
		}
		return pf, err
	}
	if pf.SchemaVersion == 0 {
		pf.SchemaVersion = 1
	}
	if pf.Jump == nil {
		pf.Jump = map[string]string{}
	}
	if pf.Run == nil {
		pf.Run = map[string]string{}
	}
	if pf.Projects == nil {
		pf.Projects = map[string]Project{}
	}
	if pf.Tags == nil {
		pf.Tags = []string{}
	}
	if pf.Examples == nil {
		pf.Examples = []string{}
	}
	return pf, nil
}

func SavePackFile(path string, pf PackFile) error {
	if pf.SchemaVersion == 0 {
		pf.SchemaVersion = 1
	}
	if pf.Jump == nil {
		pf.Jump = map[string]string{}
	}
	if pf.Run == nil {
		pf.Run = map[string]string{}
	}
	if pf.Projects == nil {
		pf.Projects = map[string]Project{}
	}
	if pf.Tags == nil {
		pf.Tags = []string{}
	}
	if pf.Examples == nil {
		pf.Examples = []string{}
	}
	return writeJSON(path, pf)
}

func readJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func writeJSON(path string, v any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func copyDirRecursive(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}
