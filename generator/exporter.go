package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DocumentData holds a single specification document's name, title, and content
type DocumentData struct {
	FileName string `json:"file_name"`
	Title    string `json:"title"`
	Content  string `json:"content"`
}

// ExportMetadata holds metadata context for the export
type ExportMetadata struct {
	ProjectName string            `json:"project_name"`
	ExportTime  string            `json:"export_time"`
	Version     string            `json:"version"`
	Metrics     interface{}       `json:"metrics,omitempty"`
	Scores      map[string]int    `json:"scores,omitempty"`
}

// ExportToHTML scans the output directory for markdown files and compiles them into a standalone index.html
func ExportToHTML(projectName string, outputDir string, distDir string) (string, error) {
	// 1. Locate files in the output directory
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return "", fmt.Errorf("output directory does not exist: %s", outputDir)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to read output directory: %w", err)
	}

	var documents []DocumentData
	var metaData []byte

	// 2. Read each document and the metadata json
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(outputDir, file.Name())

		if file.Name() == ".synthspec-meta.json" {
			data, err := os.ReadFile(filePath)
			if err == nil {
				metaData = data
			}
			continue
		}

		if strings.HasSuffix(file.Name(), ".md") {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", file.Name(), err)
			}

			// Extract title from the first header line (e.g. "# Title")
			title := file.Name()
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "# ") {
					title = strings.TrimPrefix(trimmed, "# ")
					break
				}
			}

			documents = append(documents, DocumentData{
				FileName: file.Name(),
				Title:    title,
				Content:  string(content),
			})
		}
	}

	if len(documents) == 0 {
		return "", fmt.Errorf("no markdown specification documents found in %s", outputDir)
	}

	// Sort documents by file name
	for i := 0; i < len(documents); i++ {
		for j := i + 1; j < len(documents); j++ {
			if documents[i].FileName > documents[j].FileName {
				documents[i], documents[j] = documents[j], documents[i]
			}
		}
	}

	// 3. Compile metadata
	var telemetryMeta TelemetryMetadata
	if len(metaData) > 0 {
		_ = json.Unmarshal(metaData, &telemetryMeta)
	}

	exportMeta := ExportMetadata{
		ProjectName: projectName,
		ExportTime:  time.Now().Format("2006-01-02 15:04:05"),
		Version:     "1.0.0",
	}
	if telemetryMeta.ProjectName != "" {
		exportMeta.Scores = telemetryMeta.ComplianceSummary
		exportMeta.Metrics = telemetryMeta.CompletionMetrics
	}

	docsJSON, err := json.Marshal(documents)
	if err != nil {
		return "", fmt.Errorf("failed to marshal documents to JSON: %w", err)
	}

	metaJSON, err := json.Marshal(exportMeta)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}

	// 4. Generate HTML template
	htmlContent := getHTMLTemplate(projectName, string(docsJSON), string(metaJSON))

	// 5. Ensure dist directory exists and write index.html
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(distDir, "index.html")
	if err := os.WriteFile(destPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write export HTML file: %w", err)
	}

	return destPath, nil
}

func getHTMLTemplate(projectName, docsJSON, metaJSON string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + projectName + ` - SynthSpec Solution Specifications</title>
    
    <!-- Google Fonts -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Outfit:wght@400;500;600;700&family=Fira+Code:wght@400;500&display=swap" rel="stylesheet">
    
    <!-- FontAwesome Icons -->
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    
    <!-- Prism.js Syntax Highlighting -->
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism-tomorrow.min.css">
    
    <!-- Marked Markdown Parser -->
    <script src="https://cdn.jsdelivr.net/npm/marked/marked.min.js"></script>
    
    <!-- Mermaid.js for Diagrams -->
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    
    <style>
        :root {
            --bg-color-dark: #09090b;
            --sidebar-bg-dark: rgba(20, 20, 25, 0.6);
            --content-bg-dark: rgba(24, 24, 28, 0.4);
            --border-color-dark: rgba(255, 255, 255, 0.08);
            --text-primary-dark: #f4f4f5;
            --text-secondary-dark: #a1a1aa;
            --accent-glow: rgba(99, 102, 241, 0.15);
            --accent-color: #6366f1;
            --accent-hover: #4f46e5;
            
            --bg-color-light: #fafafa;
            --sidebar-bg-light: rgba(244, 244, 245, 0.85);
            --content-bg-light: rgba(255, 255, 255, 0.9);
            --border-color-light: rgba(0, 0, 0, 0.08);
            --text-primary-light: #18181b;
            --text-secondary-light: #52525b;
            
            --transition-speed: 0.3s;
        }

        body {
            margin: 0;
            padding: 0;
            font-family: 'Inter', sans-serif;
            background-color: var(--bg-color-dark);
            color: var(--text-primary-dark);
            display: flex;
            height: 100vh;
            overflow: hidden;
            transition: background-color var(--transition-speed), color var(--transition-speed);
        }

        body.light-mode {
            background-color: var(--bg-color-light);
            color: var(--text-primary-light);
        }

        /* Ambient Glow backgrounds */
        .ambient-glow {
            position: absolute;
            width: 40vw;
            height: 40vw;
            border-radius: 50%;
            background: radial-gradient(circle, var(--accent-glow) 0%, rgba(99, 102, 241, 0) 70%);
            top: -10vw;
            right: -10vw;
            z-index: -1;
            pointer-events: none;
            filter: blur(80px);
        }

        /* Sidebar Container */
        .sidebar {
            width: 320px;
            background-color: var(--sidebar-bg-dark);
            border-right: 1px solid var(--border-color-dark);
            display: flex;
            flex-direction: column;
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            z-index: 10;
            transition: background-color var(--transition-speed), border-color var(--transition-speed);
        }

        .light-mode .sidebar {
            background-color: var(--sidebar-bg-light);
            border-right: 1px solid var(--border-color-light);
        }

        .sidebar-header {
            padding: 24px;
            border-bottom: 1px solid var(--border-color-dark);
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .light-mode .sidebar-header {
            border-bottom: 1px solid var(--border-color-light);
        }

        .brand {
            font-family: 'Outfit', sans-serif;
            font-size: 22px;
            font-weight: 700;
            letter-spacing: -0.5px;
            background: linear-gradient(135deg, #a78bfa 0%, #6366f1 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .brand i {
            -webkit-text-fill-color: #6366f1;
        }

        .project-title {
            font-size: 14px;
            color: var(--text-secondary-dark);
            font-weight: 500;
        }

        .light-mode .project-title {
            color: var(--text-secondary-light);
        }

        /* Search input */
        .search-container {
            padding: 16px 24px;
            position: relative;
        }

        .search-box {
            width: 100%;
            padding: 10px 14px 10px 36px;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid var(--border-color-dark);
            border-radius: 8px;
            color: var(--text-primary-dark);
            font-family: inherit;
            font-size: 14px;
            box-sizing: border-box;
            outline: none;
            transition: all 0.2s ease;
        }

        .light-mode .search-box {
            background: rgba(0, 0, 0, 0.03);
            border: 1px solid var(--border-color-light);
            color: var(--text-primary-light);
        }

        .search-box:focus {
            border-color: var(--accent-color);
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.25);
        }

        .search-container i {
            position: absolute;
            left: 36px;
            top: 29px;
            color: var(--text-secondary-dark);
            font-size: 14px;
        }

        .light-mode .search-container i {
            color: var(--text-secondary-light);
        }

        /* Document list */
        .nav-list {
            list-style: none;
            padding: 0;
            margin: 0;
            overflow-y: auto;
            flex-grow: 1;
        }

        .nav-item {
            margin: 4px 16px;
        }

        .nav-link {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            color: var(--text-secondary-dark);
            text-decoration: none;
            font-size: 14px;
            font-weight: 500;
            border-radius: 8px;
            transition: all 0.2s ease;
            cursor: pointer;
        }

        .light-mode .nav-link {
            color: var(--text-secondary-light);
        }

        .nav-link:hover {
            color: var(--text-primary-dark);
            background: rgba(255, 255, 255, 0.03);
        }

        .light-mode .nav-link:hover {
            color: var(--text-primary-light);
            background: rgba(0, 0, 0, 0.02);
        }

        .nav-link.active {
            color: #ffffff;
            background: var(--accent-color);
            box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
        }

        .light-mode .nav-link.active {
            color: #ffffff;
            background: var(--accent-color);
        }

        .nav-link .file-icon {
            font-size: 16px;
            width: 20px;
            text-align: center;
        }

        /* Sidebar Footer / Controls */
        .sidebar-footer {
            padding: 20px 24px;
            border-top: 1px solid var(--border-color-dark);
            display: flex;
            align-items: center;
            justify-content: space-between;
            font-size: 12px;
            color: var(--text-secondary-dark);
        }

        .light-mode .sidebar-footer {
            border-top: 1px solid var(--border-color-light);
            color: var(--text-secondary-light);
        }

        .theme-toggle {
            background: none;
            border: none;
            color: var(--text-secondary-dark);
            cursor: pointer;
            font-size: 16px;
            padding: 8px;
            border-radius: 50%;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .light-mode .theme-toggle {
            color: var(--text-secondary-light);
        }

        .theme-toggle:hover {
            background: rgba(255, 255, 255, 0.05);
            color: var(--text-primary-dark);
        }

        .light-mode .theme-toggle:hover {
            background: rgba(0, 0, 0, 0.05);
            color: var(--text-primary-light);
        }

        /* Main Content Container */
        .content-container {
            flex-grow: 1;
            height: 100%;
            display: flex;
            flex-direction: column;
            overflow: hidden;
            position: relative;
        }

        .content-header {
            height: 70px;
            border-bottom: 1px solid var(--border-color-dark);
            padding: 0 40px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            backdrop-filter: blur(12px);
            -webkit-backdrop-filter: blur(12px);
            z-index: 5;
        }

        .light-mode .content-header {
            border-bottom: 1px solid var(--border-color-light);
        }

        .doc-title-container {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .doc-title-container h2 {
            margin: 0;
            font-family: 'Outfit', sans-serif;
            font-size: 20px;
            font-weight: 600;
        }

        .doc-meta-badge {
            font-size: 11px;
            background: rgba(99, 102, 241, 0.15);
            color: var(--accent-color);
            padding: 4px 8px;
            border-radius: 12px;
            font-family: 'Fira Code', monospace;
        }

        .score-badges {
            display: flex;
            gap: 12px;
        }

        .score-badge {
            background: rgba(255, 255, 255, 0.03);
            border: 1px solid var(--border-color-dark);
            padding: 6px 12px;
            border-radius: 8px;
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 12px;
        }

        .light-mode .score-badge {
            background: rgba(0, 0, 0, 0.02);
            border: 1px solid var(--border-color-light);
        }

        .score-badge.passed {
            border-color: rgba(34, 197, 94, 0.4);
            color: #22c55e;
            background: rgba(34, 197, 94, 0.05);
        }

        /* Document Viewer Body */
        .doc-viewer {
            flex-grow: 1;
            padding: 40px;
            overflow-y: auto;
            box-sizing: border-box;
            background-color: var(--content-bg-dark);
            backdrop-filter: blur(8px);
            -webkit-backdrop-filter: blur(8px);
            transition: background-color var(--transition-speed);
        }

        .light-mode .doc-viewer {
            background-color: var(--content-bg-light);
        }

        /* Markdown Styles */
        .markdown-body {
            max-width: 850px;
            margin: 0 auto;
            line-height: 1.7;
            font-size: 16px;
        }

        .markdown-body h1 {
            font-family: 'Outfit', sans-serif;
            font-size: 32px;
            margin-top: 0;
            margin-bottom: 24px;
            font-weight: 700;
            letter-spacing: -0.5px;
            border-bottom: 1px solid var(--border-color-dark);
            padding-bottom: 12px;
        }

        .light-mode .markdown-body h1 {
            border-bottom: 1px solid var(--border-color-light);
        }

        .markdown-body h2 {
            font-family: 'Outfit', sans-serif;
            font-size: 22px;
            margin-top: 36px;
            margin-bottom: 16px;
            font-weight: 600;
            border-bottom: 1px solid rgba(255, 255, 255, 0.05);
            padding-bottom: 8px;
        }

        .light-mode .markdown-body h2 {
            border-bottom: 1px solid rgba(0, 0, 0, 0.05);
        }

        .markdown-body h3 {
            font-family: 'Outfit', sans-serif;
            font-size: 18px;
            margin-top: 24px;
            margin-bottom: 12px;
            font-weight: 600;
        }

        .markdown-body p, .markdown-body li {
            color: var(--text-primary-dark);
            transition: color var(--transition-speed);
        }

        .light-mode .markdown-body p, .light-mode .markdown-body li {
            color: var(--text-primary-light);
        }

        .markdown-body p {
            margin-bottom: 16px;
        }

        .markdown-body ul, .markdown-body ol {
            padding-left: 24px;
            margin-bottom: 16px;
        }

        .markdown-body li {
            margin-bottom: 8px;
        }

        .markdown-body code {
            font-family: 'Fira Code', monospace;
            background-color: rgba(255, 255, 255, 0.06);
            padding: 3px 6px;
            border-radius: 4px;
            font-size: 14px;
        }

        .light-mode .markdown-body code {
            background-color: rgba(0, 0, 0, 0.05);
        }

        .markdown-body pre {
            background-color: #1e1e24 !important;
            padding: 20px;
            border-radius: 12px;
            overflow-x: auto;
            border: 1px solid var(--border-color-dark);
            margin-bottom: 24px;
        }

        .light-mode .markdown-body pre {
            border: 1px solid var(--border-color-light);
        }

        .markdown-body pre code {
            background-color: transparent;
            padding: 0;
            border-radius: 0;
            color: #e4e4e7;
        }

        .markdown-body blockquote {
            margin: 24px 0;
            padding: 12px 24px;
            border-left: 4px solid var(--accent-color);
            background: rgba(99, 102, 241, 0.05);
            border-radius: 0 8px 8px 0;
        }

        .markdown-body blockquote p {
            margin: 0;
            font-style: italic;
        }

        .markdown-body table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 24px;
            font-size: 14px;
        }

        .markdown-body th, .markdown-body td {
            padding: 12px 16px;
            border: 1px solid var(--border-color-dark);
            text-align: left;
        }

        .light-mode .markdown-body th, .light-mode .markdown-body td {
            border: 1px solid var(--border-color-light);
        }

        .markdown-body th {
            background-color: rgba(255, 255, 255, 0.02);
            font-weight: 600;
        }

        .light-mode .markdown-body th {
            background-color: rgba(0, 0, 0, 0.02);
        }

        .markdown-body tr:nth-child(even) td {
            background-color: rgba(255, 255, 255, 0.01);
        }

        .light-mode .markdown-body tr:nth-child(even) td {
            background-color: rgba(0, 0, 0, 0.01);
        }

        .markdown-body hr {
            height: 1px;
            border: none;
            background-color: var(--border-color-dark);
            margin: 36px 0;
        }

        .light-mode .markdown-body hr {
            background-color: var(--border-color-light);
        }

        /* Search highlights */
        .highlight-match {
            background-color: rgba(245, 158, 11, 0.3);
            border-bottom: 2px solid #f59e0b;
            color: inherit;
        }

        /* Mermaid Wrapper */
        .mermaid-container {
            background-color: #1e1e24;
            border: 1px solid var(--border-color-dark);
            border-radius: 12px;
            padding: 24px;
            margin: 24px 0;
            display: flex;
            justify-content: center;
            overflow-x: auto;
        }

        .light-mode .mermaid-container {
            background-color: #f8f9fa;
            border: 1px solid var(--border-color-light);
        }

        /* Scrollbars styling */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }

        ::-webkit-scrollbar-track {
            background: transparent;
        }

        ::-webkit-scrollbar-thumb {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 4px;
        }

        .light-mode ::-webkit-scrollbar-thumb {
            background: rgba(0, 0, 0, 0.1);
        }

        ::-webkit-scrollbar-thumb:hover {
            background: rgba(255, 255, 255, 0.2);
        }
    </style>
</head>
<body>
    <div class="ambient-glow"></div>
    
    <!-- Sidebar Navigation -->
    <div class="sidebar">
        <div class="sidebar-header">
            <div class="brand">
                <i class="fa-solid fa-layer-group"></i>
                <span>SynthSpec</span>
            </div>
            <div class="project-title" id="projectName">` + projectName + `</div>
        </div>
        
        <div class="search-container">
            <input type="text" class="search-box" id="searchInput" placeholder="Search specifications...">
            <i class="fa-solid fa-magnifying-glass"></i>
        </div>
        
        <ul class="nav-list" id="navList">
            <!-- Dynamically populated -->
        </ul>
        
        <div class="sidebar-footer">
            <span id="exportTime">Exported: -</span>
            <button class="theme-toggle" id="themeToggle" title="Toggle Light/Dark Theme">
                <i class="fa-solid fa-sun" id="themeIcon"></i>
            </button>
        </div>
    </div>
    
    <!-- Main Content Panel -->
    <div class="content-container">
        <div class="content-header">
            <div class="doc-title-container">
                <h2 id="currentDocTitle">Select a Document</h2>
                <span class="doc-meta-badge" id="currentDocFilename">-</span>
            </div>
            <div class="score-badges" id="scoreBadges">
                <!-- Dynamically populated with compliance scores if metadata exists -->
            </div>
        </div>
        
        <div class="doc-viewer" id="docViewer">
            <div class="markdown-body" id="markdownContent">
                <!-- Rendered markdown goes here -->
            </div>
        </div>
    </div>

    <!-- Prism JS for syntax highlight -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js"></script>

    <script>
        // Injected data payload
        const docs = ` + docsJSON + `;
        const meta = ` + metaJSON + `;

        // State variables
        let activeDocIndex = 0;
        let searchQuery = "";

        // DOM elements
        const navList = document.getElementById('navList');
        const markdownContent = document.getElementById('markdownContent');
        const currentDocTitle = document.getElementById('currentDocTitle');
        const currentDocFilename = document.getElementById('currentDocFilename');
        const searchInput = document.getElementById('searchInput');
        const themeToggle = document.getElementById('themeToggle');
        const themeIcon = document.getElementById('themeIcon');
        const exportTimeText = document.getElementById('exportTime');
        const scoreBadges = document.getElementById('scoreBadges');

        // Initialize Mermaid
        mermaid.initialize({
            startOnLoad: false,
            theme: 'dark',
            securityLevel: 'loose'
        });

        // Initialize App
        function init() {
            // Set export time
            if (meta.export_time) {
                exportTimeText.textContent = "Exported: " + meta.export_time.split(" ")[0];
            }

            // Populate scores if available
            if (meta.scores) {
                scoreBadges.innerHTML = "";
                Object.entries(meta.scores).forEach(([key, val]) => {
                    const badge = document.createElement('div');
                    badge.className = 'score-badge';
                    const keyLabel = key.charAt(0).toUpperCase() + key.slice(1);
                    badge.innerHTML = '<i class="fa-solid fa-shield-halved"></i> <span>' + keyLabel + ': ' + val + '%</span>';
                    if (val >= 80) badge.classList.add('passed');
                    scoreBadges.appendChild(badge);
                });
            }

            renderSidebar();
            if (docs.length > 0) {
                loadDocument(0);
            }
        }

        // Generate dynamic file icons based on document index/pattern
        function getIconClass(filename) {
            if (filename.includes('compliance')) return 'fa-solid fa-square-check';
            if (filename.includes('domain')) return 'fa-solid fa-sitemap';
            if (filename.includes('prd')) return 'fa-solid fa-list-check';
            if (filename.includes('system_architecture')) return 'fa-solid fa-cubes';
            if (filename.includes('api_architecture')) return 'fa-solid fa-network-wired';
            if (filename.includes('coding_standards')) return 'fa-solid fa-code';
            if (filename.includes('security')) return 'fa-solid fa-user-shield';
            if (filename.includes('roadmap')) return 'fa-solid fa-route';
            return 'fa-regular fa-file-lines';
        }

        // Render sidebar items
        function renderSidebar() {
            navList.innerHTML = "";
            docs.forEach((doc, idx) => {
                // If searching, only show matched files
                if (searchQuery) {
                    const matchedTitle = doc.title.toLowerCase().includes(searchQuery.toLowerCase());
                    const matchedContent = doc.content.toLowerCase().includes(searchQuery.toLowerCase());
                    if (!matchedTitle && !matchedContent) return;
                }

                const li = document.createElement('li');
                li.className = 'nav-item';
                
                const a = document.createElement('a');
                a.className = 'nav-link' + (idx === activeDocIndex ? ' active' : '');
                
                const icon = document.createElement('i');
                icon.className = 'file-icon ' + getIconClass(doc.fileName);
                
                const textSpan = document.createElement('span');
                textSpan.textContent = doc.title;

                a.appendChild(icon);
                a.appendChild(textSpan);
                a.onclick = () => loadDocument(idx);

                li.appendChild(a);
                navList.appendChild(li);
            });
        }

        // Highlight matching query text in HTML content (basic implementation)
        function applySearchHighlight(htmlText, query) {
            if (!query) return htmlText;
            // Create regex ignoring HTML tags
            const escapedQuery = query.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
            // Match words but skip those inside HTML tags
            const regex = new RegExp("(?![^<>]*>)(" + escapedQuery + ")", "gi");
            return htmlText.replace(regex, '<span class="highlight-match">$1</span>');
        }

        // Load document and parse Markdown
        function loadDocument(index) {
            activeDocIndex = index;
            const doc = docs[index];
            if (!doc) return;

            currentDocTitle.textContent = doc.title;
            currentDocFilename.textContent = doc.fileName;

            // Render Markdown using marked.js
            let parsedHTML = marked.parse(doc.content);

            // Pre-process markdown HTML codeblocks that contain mermaid code to prevent prism parsing
            const parser = new DOMParser();
            const docDOM = parser.parseFromString(parsedHTML, 'text/html');
            const codeBlocks = docDOM.querySelectorAll('pre code');
            
            codeBlocks.forEach(block => {
                if (block.classList.contains('language-mermaid')) {
                    const mermaidCode = block.textContent;
                    const container = document.createElement('div');
                    container.className = 'mermaid-container';
                    const div = document.createElement('div');
                    div.className = 'mermaid';
                    div.textContent = mermaidCode;
                    container.appendChild(div);
                    block.parentElement.replaceWith(container);
                }
            });

            parsedHTML = docDOM.body.innerHTML;

            // Apply search highlighting to the text
            if (searchQuery) {
                parsedHTML = applySearchHighlight(parsedHTML, searchQuery);
            }

            markdownContent.innerHTML = parsedHTML;

            // Refresh Prism syntax highlighting
            Prism.highlightAllUnder(markdownContent);

            // Render Mermaid diagrams
            try {
                mermaid.run({
                    nodes: docDOM.querySelectorAll('.mermaid')
                });
            } catch (e) {
                console.error("Mermaid parsing failed: ", e);
            }

            // Update sidebar active states
            const navLinks = navList.querySelectorAll('.nav-link');
            navLinks.forEach((link, idx) => {
                if (idx === index) {
                    link.classList.add('active');
                } else {
                    link.classList.remove('active');
                }
            });
            
            // Scroll back to top
            document.getElementById('docViewer').scrollTop = 0;
        }

        // Theme Toggle
        themeToggle.addEventListener('click', () => {
            const isLight = document.body.classList.toggle('light-mode');
            themeIcon.className = isLight ? 'fa-solid fa-moon' : 'fa-solid fa-sun';
            
            // Re-initialize mermaid with new theme
            const currentTheme = isLight ? 'default' : 'dark';
            mermaid.initialize({
                theme: currentTheme
            });
            
            // Re-render to update diagram theme colors
            loadDocument(activeDocIndex);
        });

        // Search Input handler
        searchInput.addEventListener('input', (e) => {
            searchQuery = e.target.value;
            renderSidebar();
            // Re-render current page to show highlights
            loadDocument(activeDocIndex);
        });

        // Run
        init();
    </script>
</body>
</html>`
}
