FROM node:20-alpine
WORKDIR /app

COPY package*.json ./

RUN npm ci

COPY . .

RUN addgroup -S mercari && adduser -S trainee -G mercari

RUN chown -R trainee:mercari /app/node_modules && \
    chmod -R 755 /app/node_modules

EXPOSE 3000

USER trainee

CMD ["npm", "start", "--", "--host", "0.0.0.0"]