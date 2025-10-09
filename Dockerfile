# ---- Builder Stage ----
# This stage installs dependencies, including build-time tools.
FROM python:3.9-alpine AS builder

# Install OS-level dependencies needed to build opencv-python
RUN apk add --no-cache build-base cmake linux-headers libjpeg-turbo-dev libpng-dev tiff-dev

WORKDIR /app

# Copy requirements and install Python packages
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt


# ---- Final Stage ----
# This stage creates the lean, final image for runtime.
FROM python:3.9-alpine

WORKDIR /app

# Install only the runtime OS dependencies for OpenCV
RUN apk add --no-cache libjpeg-turbo libpng tiff

# Copy installed Python packages from the builder stage
COPY --from=builder /usr/local/lib/python3.9/site-packages /usr/local/lib/python3.9/site-packages

# Copy the application code
COPY ShapeDetection.py .

# Expose the port Gunicorn will run on
EXPOSE 8000

# Command to run the application using Gunicorn
CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "4", "ShapeDetection:app"]