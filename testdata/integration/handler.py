from flask import Flask, request, jsonify
from functools import wraps

app = Flask(__name__)


def require_auth(f):
    @wraps(f)
    def decorated(*args, **kwargs):
        token = request.headers.get("Authorization")
        if not token:
            return jsonify({"error": "Missing token"}), 401
        return f(*args, **kwargs)
    return decorated


@app.route("/health")
def health():
    return jsonify({"status": "ok"})


@app.route("/items", methods=["GET", "POST"])
@require_auth
def items():
    if request.method == "POST":
        data = request.get_json()
        return jsonify({"created": data}), 201
    return jsonify({"items": ["a", "b", "c"]})


if __name__ == "__main__":
    app.run(debug=True)
