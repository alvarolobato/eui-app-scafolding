FROM --platform=$BUILDPLATFORM node:20-alpine AS build
ARG TARGETPLATFORM
ARG BUILDPLATFORM

WORKDIR /app
COPY package.json yarn.lock /app/
RUN yarn install
ADD . /app
RUN npm run build

FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
