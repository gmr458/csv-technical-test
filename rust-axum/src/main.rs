use std::{
    collections::HashMap,
    sync::{Arc, RwLock},
};

use axum::{
    extract::{MatchedPath, Multipart, Query, State},
    http::{Request, StatusCode},
    response::IntoResponse,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use serde_json::json;
use tower_http::trace::TraceLayer;
use tracing::info_span;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

#[tokio::main]
async fn main() {
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| {
                "csv_technical_test_rust_axum=debug,tower_http=debug,axum::rejection=trace".into()
            }),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    let db = GlobalData::default();

    let app = Router::new()
        .route("/api/files", post(handler_files))
        .route("/api/users", get(handler_users))
        .layer(
            TraceLayer::new_for_http().make_span_with(|request: &Request<_>| {
                let matched_path = request
                    .extensions()
                    .get::<MatchedPath>()
                    .map(MatchedPath::as_str);

                info_span!(
                    "http_request",
                    method = ?request.method(),
                    matched_path,
                    some_other_field = tracing::field::Empty,
                )
            }),
        )
        .with_state(db);

    let listener = tokio::net::TcpListener::bind("127.0.0.1:3000")
        .await
        .unwrap();

    println!("listening on port {}", listener.local_addr().unwrap());
    axum::serve(listener, app).await.unwrap();
}

type GlobalData = Arc<RwLock<Vec<HashMap<String, String>>>>;

#[derive(Serialize)]
struct ResponseMessage {
    message: String,
}

#[derive(Serialize)]
struct ResponseData {
    data: Option<Vec<HashMap<String, String>>>,
    error: Option<String>,
}

async fn handler_files(
    State(db): State<GlobalData>,
    mut multipart: Multipart,
) -> impl IntoResponse {
    while let Ok(field) = multipart.next_field().await {
        match field {
            Some(field) => {
                if let Some(name) = field.name() {
                    if name != "file" {
                        continue;
                    }
                } else {
                    return (
                        StatusCode::INTERNAL_SERVER_ERROR,
                        Json(json!({"error": "Internal server error"})),
                    );
                }

                if let Some(content_type) = field.content_type() {
                    if content_type != "text/csv" {
                        return (
                            StatusCode::BAD_REQUEST,
                            Json(json!({"message": "The file type must be CSV"})),
                        );
                    }
                }

                match field.text().await {
                    Ok(text) => {
                        if text.len() == 0 {
                            return (
                                StatusCode::BAD_REQUEST,
                                Json(json!({"message": "The file must not be empty"})),
                            );
                        }

                        let mut lines = text.trim().lines();

                        let keys = match lines.next() {
                            Some(keys) => keys,
                            None => {
                                return (
                                    StatusCode::BAD_REQUEST,
                                    Json(json!({"message": "The file must not be empty"})),
                                );
                            }
                        };
                        let keys: Vec<&str> = keys.split(',').collect();

                        let rows: Vec<Vec<&str>> =
                            lines.map(|line| line.split(',').collect()).collect();

                        if rows.len() == 0 {
                            return (
                                StatusCode::BAD_REQUEST,
                                Json(json!({"message": "Send a file with records"})),
                            );
                        }

                        let mut db = match db.write() {
                            Ok(db) => db,
                            Err(err) => {
                                eprintln!("{}", err);
                                return (
                                    StatusCode::INTERNAL_SERVER_ERROR,
                                    Json(json!({"message": "Internal server error"})),
                                );
                            }
                        };

                        if db.len() > 0 {
                            db.clear();
                        }

                        for cell in rows {
                            let mut item: HashMap<String, String> = HashMap::new();
                            for (index, key) in keys.iter().enumerate() {
                                item.insert(key.to_string(), cell[index].to_string());
                            }
                            db.push(item);
                        }

                        return (
                            StatusCode::OK,
                            Json(json!({"message": "File uploaded successfully"})),
                        );
                    }
                    Err(err) => {
                        eprintln!("{}", err);
                        return (
                            StatusCode::INTERNAL_SERVER_ERROR,
                            Json(json!({"error": "Internal server error"})),
                        );
                    }
                }
            }
            None => break,
        }
    }

    return (
        StatusCode::BAD_REQUEST,
        Json(json!({"error": "A file must be provided"})),
    );
}

#[derive(Deserialize)]
struct Search {
    q: Option<String>,
}

async fn handler_users(State(db): State<GlobalData>, search: Query<Search>) -> impl IntoResponse {
    match db.read() {
        Ok(db) => {
            let data = db.clone();
            if data.len() == 0 {
                return (
                    StatusCode::OK,
                    Json(json!({
                        "message": "There is not data, upload a CSV file first",
                    })),
                );
            }

            if let Some(q) = &search.q {
                let mut coincidences: Vec<HashMap<String, String>> = Vec::new();
                let q = q.to_lowercase();

                for record in data {
                    for (_, value) in &record {
                        if value.to_lowercase().contains(&q) {
                            coincidences.push(record.clone());
                        }
                    }
                }

                return (
                    StatusCode::OK,
                    Json(json!({
                        "data": coincidences,
                    })),
                );
            } else {
                return (
                    StatusCode::OK,
                    Json(json!({
                        "data": data,
                    })),
                );
            }
        }
        Err(err) => {
            eprintln!("{}", err);

            return (
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(json!({
                    "error": "Internal server error",
                })),
            );
        }
    }
}
