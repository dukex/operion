FROM node:20-alpine as vite-app

ARG API_BASE_URL

WORKDIR /app

COPY package.json package-lock.json ./

RUN npm install

COPY . .

ENV API_BASE_URL=${API_BASE_URL}

RUN  npm run build

FROM nginx:alpine

WORKDIR /usr/share/nginx/

RUN rm -rf html && mkdir html

WORKDIR /

COPY ./nginx/nginx.conf /etc/nginx
COPY --from=vite-app ./app/dist /usr/share/nginx/html

ENTRYPOINT ["nginx", "-g", "daemon off;"]