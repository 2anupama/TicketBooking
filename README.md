# Cinema Ticket Booking System

A modern web-based cinema ticket booking system built with Go and modern web technologies.

## Features

- Browse available movies and show times
- Interactive seat selection
- Real-time booking updates
- Concurrent booking management
- User-friendly interface
- Email confirmation system
- SQLite database for easy setup

## Prerequisites

- Go 1.21 or later
- Modern web browser

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd cinema-booking
```

2. Install Go dependencies:
```bash
go mod download
```

The application uses SQLite as its database, which requires no additional setup. The database file (`cinema.db`) will be created automatically when you first run the application.

## Running the Application

1. Start the server:
```bash
go run 2447212.go
```

2. Open your browser and navigate to:
```
http://localhost:8080
```

The application comes with pre-seeded data including:
- Three movies (The Matrix, Inception, Interstellar)
- Multiple show times for each movie
- Seat layouts (8x8 grid) for each show

## API Endpoints

- `GET /api/movies` - Get all available movies
- `GET /api/seats?showId=<id>` - Get seats for a specific show
- `POST /api/book` - Book tickets

## Testing

Run the tests with:
```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details. # TicketBooking
# BookingProject
