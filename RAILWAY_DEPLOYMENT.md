# Railway Deployment Guide

This guide will help you deploy your Go backend to Railway with PostgreSQL database.

## 🚀 Quick Setup

### Step 1: Prepare Your Repository

1. **Commit all changes** to your backend repository:
   ```bash
   git add .
   git commit -m "Prepare for Railway deployment"
   git push origin main
   ```

### Step 2: Deploy to Railway

1. **Go to [Railway.app](https://railway.app)** and sign up/login
2. **Create New Project** → **Deploy from GitHub repo**
3. **Connect your GitHub account** and select your backend repository
4. **Railway will automatically detect** your Go project and use the Dockerfile

### Step 3: Add PostgreSQL Database

1. In your Railway project dashboard, click **"+ New Service"**
2. Select **"Database"** → **"PostgreSQL"**
3. Railway will automatically create the database and provide connection details

### Step 4: Configure Environment Variables

In Railway dashboard, go to **Variables** tab and add these environment variables:

#### Required Variables:
```bash
# Application
APP_ENV=production
APP_NAME=laps
HTTP_PORT=8080

# JWT Security - CHANGE THIS!
JWT_SIGNING_KEY=your-super-secret-jwt-key-change-this-now-12345
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=24h

# Database (Railway auto-populates these, but you can override)
POSTGRES_SSL_MODE=require
POSTGRES_MAX_CONNECTIONS=10

# CORS - Update with your Vercel domain
CORS_ALLOWED_ORIGINS=https://your-app.vercel.app,http://localhost:3000
```

#### Optional S3 Variables (if using file uploads):
```bash
S3_ENDPOINT=your-s3-endpoint
S3_REGION=us-east-1
S3_ACCESS_KEY_ID=your-access-key
S3_SECRET_ACCESS_KEY=your-secret-key
S3_BUCKET=laps
S3_USE_SSL=true
```

### Step 5: Get Your API URL

1. After deployment, Railway will provide a URL like: `https://your-app-name.railway.app`
2. Your API will be available at: `https://your-app-name.railway.app/api/v1`
3. Swagger docs: `https://your-app-name.railway.app/swagger`

## 🔧 Update Your Frontend

Update your frontend environment variables to point to Railway:

```bash
# In your Vercel project settings
NEXT_PUBLIC_API_URL=https://your-app-name.railway.app/api/v1
NEXT_PUBLIC_WS_URL=wss://your-app-name.railway.app/ws
```

## 📊 Monitoring & Logs

- **View Logs**: Railway dashboard → Your service → Logs tab
- **Monitor Performance**: Railway dashboard → Metrics tab
- **Database Management**: Railway dashboard → PostgreSQL service

## 💰 Costs

- **Hobby Plan**: $5/month (covers app + database)
- **Includes**: 512MB RAM, shared CPU, 1GB disk
- **Perfect for**: Development and small production apps

## 🔧 Troubleshooting

### Common Issues:

1. **Build Fails**:
   - Check that your `go.mod` and `go.sum` are committed
   - Verify Dockerfile syntax

2. **Database Connection Issues**:
   - Ensure `POSTGRES_SSL_MODE=require` for production
   - Check that Railway database variables are set

3. **CORS Issues**:
   - Update `CORS_ALLOWED_ORIGINS` with your Vercel domain
   - Don't use `*` in production

4. **Environment Variables**:
   - Double-check all required variables are set
   - Restart deployment after changing variables

### Health Check:
Test your deployment:
```bash
curl https://your-app-name.railway.app/api/v1/test-no-auth
```

Should return: `{"message": "no auth required", "path": "/test-no-auth"}`

## 🔄 Automatic Deployments

Railway automatically deploys when you push to your main branch. To disable:
1. Go to Settings → Service → Deployments
2. Turn off "Auto Deploy"

## 🎯 Next Steps

1. **Update frontend** to use Railway API URL
2. **Test all endpoints** work correctly
3. **Set up custom domain** (optional)
4. **Monitor logs** for any issues
5. **Set up backup strategy** for database

## 📞 Support

- **Railway Docs**: https://docs.railway.app
- **Railway Discord**: https://discord.gg/railway
- **Railway Status**: https://status.railway.app
