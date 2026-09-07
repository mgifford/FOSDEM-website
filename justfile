year   := "2027"
host   := "www-public0.fosdem.org"
config := "hugo.landing.yaml"

default:
    @just --list

# Kerberos ticket for www-public0, lasts a few hours
ticket id:
    kinit {{id}}@FOSDEM.ORG
    klist

# Landing site, staging baseURL, drafts included
build-staging:
    hugo build --config {{config}} -b https://staging.fosdem.org/{{year}}/ -D
    validate-ics public

# Landing site, production baseURL, drafts excluded
build-live:
    hugo build --config {{config}} -b https://fosdem.org/{{year}}/
    validate-ics public

# Sync to staging. Dry run unless go=1
deploy-staging go="": build-staging
    rsync -avz --delete {{ if go == "" { "--dry-run" } else { "" } }} public/ \
        www-staging@{{host}}:/var/www/staging.fosdem.org/public/{{year}}/

# Sync to production. Dry run unless go=1
deploy-live go="": build-live
    rsync -avz --delete {{ if go == "" { "--dry-run" } else { "" } }} public/ \
        www-live@{{host}}:/var/www/fosdem.org/public/{{year}}/
