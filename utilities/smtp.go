package utilities

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-mail/mail"
)

func SendSMTPVerificationEmail(user models.User) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Please verify your account")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone created a Treningheten account using your e-mail. If this wasn't you, please ignore this e-mail.<br><br>To verify the new account, visit Treningheten and verify the account using this code: <b>"+*user.VerificationCode+"</b>.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPResetEmail(user models.User) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL + "/login?reset_code=" + *user.ResetCode

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Password reset request")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone attempted a password change on your Treningheten account. If this wasn't you, please ignore this e-mail.<br><br>To reset your password, visit Treningheten using <a href='"+link+"' target='_blank'>this link</a>.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPSundayReminderEmail(user models.User, season models.Season, timeStamp time.Time) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL
	_, weekNumber := timeStamp.ISOWeek()

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Sunday reminder")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>It's Sunday and week "+strconv.Itoa(weekNumber)+" within '"+season.Name+"' is almost over.<br><br>If you haven't already, head to Treningheten using <a href='"+link+"' target='_blank'>this link</a> and log your workouts.<br><br>You can disable this alert in your settings.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPSeasonStartEmail(season models.SeasonObject) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	for _, goal := range season.Goals {

		email, emailFound, err := database.GetUserEmailByUserID(goal.User.ID)
		if err != nil {
			log.Info("Failed to get e-mail for user. Error: " + err.Error())
			continue
		} else if !emailFound {
			log.Info("User e-mail not found. Error: " + err.Error())
			continue
		}

		if strings.ToLower(config.TreninghetenEnvironment) == "test" {
			email = config.TreninghetenTestEmail
		}

		log.Info("Sending e-mail to: " + email + ".")

		link := config.TreninghetenExternalURL

		m := mail.NewMessage()
		m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
		m.SetHeader("To", email)
		m.SetHeader("Subject", "A new season has begun")
		m.SetBody("text/html", "Hello <b>"+goal.User.FirstName+"</b>!<br><br>It's Monday and a new season of Treningheten has begun. You signed up for "+strconv.Itoa(goal.ExerciseInterval)+" exercise(s) a week, and there is no going back now.<br><br>To achieve your goal you must log all exercises at the Treningheten website. Go to Treningheten using <a href='"+link+"' target='_blank'>this link</a>.")

		d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

		// Send the email
		err = d.DialAndSend(m)
		if err != nil {
			log.Info("Failed to send e-mail. Error: " + err.Error())
			continue
		}

	}

	return nil

}

func SendSMTPForWeekLost(user models.User, weekNumber int) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Your week didn't go as planned")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>You didn't hit your goal for week "+strconv.Itoa(weekNumber)+". 😢<br><br>If you haven't already, head to Treningheten using <a href='"+link+"' target='_blank'>this link</a> and check who won.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPForWheelSpin(user models.User, weekNumber int) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "You have a wheel to spin")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>You didn't hit your goal for week "+strconv.Itoa(weekNumber)+". 😢<br><br>If you haven't already, head to Treningheten using <a href='"+link+"' target='_blank'>this link</a> and spin the wheel.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPForWheelSpinCheck(user models.User, weekNumber int) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Someone spun the wheel")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone spun the wheel, check if you won in week "+strconv.Itoa(weekNumber)+". 🏆<br><br>If you haven't already, head to Treningheten using <a href='"+link+"' target='_blank'>this link</a> and check out the wheel spin.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}

func SendSMTPForWheelSpinWin(user models.User, weekNumber int) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	if strings.ToLower(config.TreninghetenEnvironment) == "test" {
		user.Email = config.TreninghetenTestEmail
	}

	log.Info("Sending e-mail to: " + user.Email + ".")

	link := config.TreninghetenExternalURL

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.TreninghetenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Someone just paid their dues")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone failed to hit their goal, and you won for week "+strconv.Itoa(weekNumber)+". 🏆<br><br>If you haven't already, head to Treningheten using <a href='"+link+"' target='_blank'>this link</a> and check out your prize.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}
