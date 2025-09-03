#!/bin/bash

# Railway Setup Script
# This script helps you set up environment variables for Railway deployment

echo "🚀 Railway Deployment Setup"
echo "=========================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Please run this script from your backend directory"
    exit 1
fi

# Generate a random JWT secret
JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || echo "please-change-this-jwt-secret-$(date +%s)")

echo ""
echo "📋 Copy these environment variables to your Railway project:"
echo "============================================================"
echo ""
echo "# Application Configuration"
echo "APP_ENV=production"
echo "APP_NAME=laps"
echo "HTTP_PORT=8080"
echo ""
echo "# JWT Configuration (🔒 KEEP SECRET!)"
echo "JWT_SIGNING_KEY=$JWT_SECRET"
echo "JWT_ACCESS_TOKEN_TTL=15m"
echo "JWT_REFRESH_TOKEN_TTL=24h"
echo ""
echo "# Database Configuration"
echo "POSTGRES_SSL_MODE=require"
echo "POSTGRES_MAX_CONNECTIONS=10"
echo "POSTGRES_MAX_IDLE_CONNECTIONS=5"
echo "POSTGRES_MAX_LIFETIME=5m"
echo ""
echo "# CORS Configuration (⚠️  UPDATE WITH YOUR VERCEL DOMAIN!)"
echo "CORS_ALLOWED_ORIGINS=https://your-app.vercel.app,http://localhost:3000"
echo ""
echo "# Optional S3 Configuration (leave empty if not using)"
echo "S3_ENDPOINT="
echo "S3_REGION=us-east-1"
echo "S3_ACCESS_KEY_ID="
echo "S3_SECRET_ACCESS_KEY="
echo "S3_BUCKET=laps"
echo "S3_USE_SSL=true"
echo ""
echo "============================================================"
echo ""
echo "📝 Next Steps:"
echo "1. Go to your Railway project dashboard"
echo "2. Click on your service → Variables tab"
echo "3. Add the variables above"
echo "4. Update CORS_ALLOWED_ORIGINS with your actual Vercel domain"
echo "5. Deploy your app!"
echo ""
echo "🔗 Useful Links:"
echo "- Railway Dashboard: https://railway.app/dashboard"
echo "- Deployment Guide: ./RAILWAY_DEPLOYMENT.md"
echo ""
