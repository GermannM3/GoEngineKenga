use std::fs;
use std::path::Path;
use tauri::Manager;

#[tauri::command]
fn import_dropped_assets(paths: Vec<String>, project_dir: String) -> Result<String, String> {
    let assets_dir = Path::new(&project_dir).join("assets");
    fs::create_dir_all(&assets_dir).map_err(|e| e.to_string())?;

    for src in &paths {
        let src_path = Path::new(src);
        if src_path.is_file() {
            let name = src_path
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("asset");
            let dest = assets_dir.join(name);
            fs::copy(src_path, &dest).map_err(|e| e.to_string())?;
        }
    }

    std::process::Command::new("kenga")
        .args(["import", "--project", &project_dir])
        .current_dir(&project_dir)
        .output()
        .map_err(|e| e.to_string())?;

    Ok(format!("Imported {} file(s)", paths.len()))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![import_dropped_assets])
        .setup(|app| {
            #[cfg(desktop)]
            let _ = app.handle().plugin(tauri_plugin_window_state::Builder::default().build());
            #[cfg(debug_assertions)]
            {
                if let Ok(window) = app.get_webview_window("main") {
                    window.open_devtools();
                }
            }
            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
