use actix_web::{body, http, web, App, HttpRequest, HttpResponse, HttpServer};
use reqwest::{Client, Method, Request};

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    // 1080番でListen
    HttpServer::new(|| {
        App::new()
            .default_service(web::to(proxy_handler))
    })
    .bind(("0.0.0.0", 1080))?
    .run()
    .await
}

async fn proxy_handler(req: HttpRequest, mut payload: web::Payload) -> actix_web::Result<HttpResponse> {
    let client = Client::default();

    // 転送先URLを構築（例: 8080に全てパススルー）
    let mut new_uri = format!("http://127.0.0.1:8080{}", req.uri());


    let new_method = match req.method() {
        &http::Method::GET => Method::GET,
        &http::Method::POST => Method::POST,
        &http::Method::PUT => Method::PUT,
        &http::Method::DELETE => Method::DELETE,
        &http::Method::PATCH => Method::PATCH,
        &http::Method::HEAD => Method::HEAD,
        &http::Method::OPTIONS => Method::OPTIONS,
        _ => return Err(actix_web::error::ErrorMethodNotAllowed("Unsupported method")),
    };

    // 新しいリクエストをビルド
    let mut forward_req = client.request(new_method, &new_uri);
    // ヘッダーを転送
    for (h_name, h_value) in req.headers().iter() {
        forward_req.header(h_name.as_str(), h_value.as_bytes());
    };

    // ボディをストリーミングで転送
    let mut body = payload.to_bytes().await.map_err(|e| {
        actix_web::error::ErrorBadRequest(format!("Failed to read request body: {:?}", e))
    })?;

    // // 転送リクエスト実行
    // let response = forward_req.send_body(body.freeze()).await.map_err(|e| {
    //     actix_web::error::ErrorBadGateway(format!("Proxy error: {:?}", e))
    // })?;

    // レスポンス組み立て
    let mut client_resp = HttpResponse::build(response.status());
    for (h_name, h_value) in response.headers().iter() {
        client_resp.append_header((h_name.clone(), h_value.clone()));
    }
    let bytes = response.body().limit(10_485_760).await?; // 10MBまで

    Ok(client_resp.body(bytes))
}