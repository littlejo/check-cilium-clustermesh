# Étape 1 : Build de l'application
FROM golang:alpine AS builder

# Crée un répertoire de travail
WORKDIR /app

# Copie le fichier source Go dans l'image
COPY ./main.go ./go.mod .

# Compile l'application en un binaire statique
RUN go build -o server .

# Étape 2 : Créer une image minimale pour exécuter le binaire
FROM alpine:latest

ENV CLUSTER="CLUSTER1"

# Copie le binaire compilé depuis l'étape précédente
COPY --from=builder /app/server /server

# Expose le port 8080
EXPOSE 8080

# Commande à exécuter au démarrage du container
CMD ["/server"]
