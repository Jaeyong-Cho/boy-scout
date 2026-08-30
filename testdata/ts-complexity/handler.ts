export function handler(event: any) {
  if (event.type === "request") {
    if (event.method === "GET") {
      if (event.path === "/api/users") {
        if (event.headers["authorization"]) {
          if (event.query.id) {
            if (event.query.id !== "") {
              return { status: 200, body: "OK" };
            }
          }
        }
      }
    }
  }
  return { status: 404, body: "Not Found" };
}
