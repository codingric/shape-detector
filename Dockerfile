# ---- Builder Stage ----
# Use a Debian-based slim image which has pre-compiled wheels for OpenCV,
# avoiding the need for a lengthy compilation process.
FROM python:3.9-slim-bookworm AS builder

WORKDIR /app

# Copy requirements and install Python packages
COPY requirements.txt .
# Using pre-compiled wheels makes this step significantly faster.
RUN pip install --no-cache-dir -r requirements.txt


# ---- Final Stage ----
# This stage creates the lean, final image for runtime.
FROM python:3.9-slim-bookworm

WORKDIR /app

# Install only the runtime OS dependencies for OpenCV
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-glx \
    && rm -rf /var/lib/apt/lists/*

# Copy installed Python packages from the builder stage
COPY --from=builder /usr/local/lib/python3.9/site-packages /usr/local/lib/python3.9/site-packages

# Copy the application code
COPY ShapeDetection.py .

# Expose the port Gunicorn will run on
EXPOSE 8000

# Command to run the application using Gunicorn
CMD ["gunicorn", "--bind", "0.0.0.0:8000", "--workers", "4", "ShapeDetection:app"]