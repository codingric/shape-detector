import cv2
import numpy as np
import json
import requests
from flask import Flask, request, jsonify

app = Flask(__name__)
app.logger.setLevel("INFO")


@app.route('/', methods=['POST'])
def detect_shapes():
    """
    Detects shapes in an image based on provided zones.
    Expects a POST request with a JSON body:
    {
        "url": "http://path.to/your/image.jpg",
        "ref": [x1, y1, x2, y2],
        "min_area": 1000,
        "zones": [
            {"name": "zone1", "region": [x1, y1, x2, y2], "min_area": 500},
            ...
        ]
    }
    """
    data = request.get_json()
    if not data or 'url' not in data or 'ref' not in data or 'zones' not in data:
        return jsonify({"error": "Invalid request body. 'url', 'ref', and 'zones' are required."}), 400

    image_url = data['url']
    ref = data['ref']
    zones = data['zones']
    global_min_area = data.get("min_area", 1000)

    try:
        # Download the image from the URL
        response = requests.get(image_url, timeout=10, verify=False)
        response.raise_for_status()  # Raise an exception for bad status codes
        image_array = np.frombuffer(response.content, np.uint8)
        image = cv2.imdecode(image_array, cv2.IMREAD_COLOR)
        if image is None:
            raise ValueError("Could not decode image.")
    except requests.exceptions.RequestException as e:
        return jsonify({"error": f"Failed to download image from URL: {e}"}), 400
    except ValueError as e:
        return jsonify({"error": str(e)}), 400

    app.logger.info("Image downloaded successfully.")

    # Convert to grayscale for thresholding
    gray_image = cv2.cvtColor(image, cv2.COLOR_BGR2GRAY)

    # Increase contrast by applying CLAHE (Contrast Limited Adaptive Histogram Equalization).
    # This makes whites whiter and blacks blacker, which helps with thresholding.
    clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8,8))
    gray_image = clahe.apply(gray_image)

    # Convert the processed grayscale image back to a 3-channel BGR image
    # so that colored rectangles can be drawn on it for visualization.
    image = cv2.cvtColor(gray_image, cv2.COLOR_GRAY2BGR)

    rx1, ry1, rx2, ry2 = ref
    ref_gray = gray_image[ry1:ry2, rx1:rx2]
    avg_pixel_value = cv2.mean(ref_gray)[0] + 5

    cv2.rectangle(image, (rx1, ry1), (rx2, ry2), (255,255,255), 5)
    app.logger.info(f"Average ref value: {avg_pixel_value}")


    resp = {}

    colours = [(0, 0, 255), (0, 255, 255), (0, 255, 0)]
    ci = 0

    # Process each zone individually
    for zone in zones:
        resp[zone["name"]] = False
        x1, y1, x2, y2 = zone["region"]
        cv2.rectangle(image, (x1, y1), (x2, y2), colours[ci], 5)

        zone_gray = gray_image[y1:y2, x1:x2]
        # Apply thresholding to the zone using its average as the threshold value
        _, zone_thresh = cv2.threshold(zone_gray, avg_pixel_value, 255, cv2.THRESH_BINARY)

        # Find contours within the thresholded zone
        contours, _ = cv2.findContours(zone_thresh, cv2.RETR_TREE, cv2.CHAIN_APPROX_SIMPLE)

        # Determine the minimum area, falling back from zone-specific to global to default
        min_area_for_zone = zone.get("min_area", global_min_area)

        for contour in contours:
            # Filter out small contours by area
            if cv2.contourArea(contour) < min_area_for_zone:
                continue

            # A more typical epsilon value for better shape approximation
            epsilon = 0.02 * cv2.arcLength(contour, True)
            approx = cv2.approxPolyDP(contour, epsilon, True)

            app.logger.info(f"Shape detected in zone {zone['name']} with area {cv2.contourArea(contour)}")
            # If a significant contour is found, mark the zone as True
            resp[zone["name"]] = True

            # Draw a rectangle around contour, save back to image
            x, y, w, h = cv2.boundingRect(approx)
            cv2.rectangle(image, (x1 + x, y1 + y), (x1 + x + w, y1 + y + h), colours[ci], 2)
            

            break # Move to the next zone once a shape is found
        ci = (ci + 1) % 3

    # save image with the timestamp in the name to /tmp
    fname = f"/tmp/{np.datetime_as_string(np.datetime64('now')).replace(':', '')}.png"
    cv2.imwrite(fname, image)
    app.logger.info(f"Image saved to {fname}")
    return jsonify(resp)

if __name__ == '__main__':
    # Example: python3 ShapeDetection.py
    # The server will run on http://127.0.0.1:5000
    app.run(debug=True)
