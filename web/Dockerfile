# Dev-focused image running the Vite dev server with HMR.
# A separate multi-stage production build can be added once hosting is decided.
FROM node:22-alpine

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install

COPY . .

EXPOSE 5173

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
